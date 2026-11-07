/**
 * Pure helpers for opt-in browser download interception.
 * Loaded via importScripts (Chrome SW) or script order (Firefox).
 */

/**
 * Returns true when a browser DownloadItem should be handed to ODM.
 * Only http(s); skips HTML/XHTML so normal navigations are not stolen.
 */
function shouldInterceptDownload(item) {
  if (!item || typeof item.url !== "string") {
    return false;
  }
  const rawUrls = [item.url];
  if (item.finalUrl && String(item.finalUrl) !== String(item.url)) {
    rawUrls.push(item.finalUrl);
  }
  let primary = null;
  for (const raw of rawUrls) {
    let url;
    try {
      url = new URL(String(raw).trim());
    } catch {
      return false;
    }
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      return false;
    }
    // YouTube / googlevideo must use the media (yt-dlp) flow — never steal
    // page navigations named "watch" / "downloads" or raw CDN stream scraps.
    if (isYouTubeRelatedHost(url.hostname)) {
      return false;
    }
    // GitHub repo/release *pages* are HTML; only asset/archive URLs are files.
    if (isGitHubNonAssetPage(url)) {
      return false;
    }
    if (!primary) {
      primary = url;
    }
  }
  if (!primary) {
    return false;
  }
  const mime = String(item.mime || "")
    .toLowerCase()
    .trim();
  // Empty mime is allowed — many file downloads omit it at onCreated time.
  if (mime.startsWith("text/html") || mime.startsWith("application/xhtml")) {
    return false;
  }
  return true;
}

/**
 * True for github.com navigations that are not real file downloads
 * (release asset, archive, or raw blob). Prevents stealing the HTML page as
 * a completed “Open-Download-Manager” task (~100–300KB).
 */
function isGitHubNonAssetPage(url) {
  const host = String(url.hostname || "")
    .toLowerCase()
    .replace(/\.$/, "");
  if (host !== "github.com" && host !== "www.github.com") {
    return false;
  }
  const path = String(url.pathname || "");
  if (path.includes("/releases/download/")) {
    return false;
  }
  if (path.includes("/archive/")) {
    return false;
  }
  if (/\/raw\//.test(path) || /\/raw\./.test(path)) {
    return false;
  }
  return true;
}

/** UUID / opaque CDN object ids used as GitHub release-asset path segments. */
function looksLikeOpaqueCDNName(name) {
  const raw = String(name || "").trim();
  if (!raw) {
    return false;
  }
  const stem = raw.includes(".") ? raw.slice(0, raw.lastIndexOf(".")) : raw;
  if (
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(
      stem,
    )
  ) {
    return true;
  }
  if (/^[0-9a-f]{32,}$/i.test(stem)) {
    return true;
  }
  return false;
}

/** Basename of a URL path (ignores query/hash). */
function urlPathBasename(raw) {
  try {
    const u = new URL(String(raw || "").trim());
    const parts = u.pathname.split("/").filter(Boolean);
    return downloadBasename(parts[parts.length - 1] || "");
  } catch {
    return "";
  }
}

/**
 * Pick a user-visible filename for an intercepted download.
 * Prefer browser filename, then the pre-redirect URL (GitHub
 * …/releases/download/tag/odm-amd64-installer.exe), never a CDN UUID.
 */
function resolveDownloadFilename(item) {
  if (!item) {
    return "";
  }
  const fromBrowser = downloadBasename(item.filename || "");
  const fromOriginal = urlPathBasename(item.url || "");
  const fromFinal = urlPathBasename(item.finalUrl || item.url || "");

  const candidates = [fromBrowser, fromOriginal, fromFinal];
  for (const c of candidates) {
    if (c && !looksLikeOpaqueCDNName(c)) {
      return c;
    }
  }
  return fromBrowser || fromOriginal || fromFinal || "";
}

/** Hosts that belong to YouTube media (watch pages + video CDN). */
function isYouTubeRelatedHost(hostname) {
  const host = String(hostname || "")
    .toLowerCase()
    .replace(/\.$/, "");
  if (
    host === "youtu.be" ||
    host === "youtube.com" ||
    host === "www.youtube.com" ||
    host === "m.youtube.com" ||
    host === "music.youtube.com" ||
    host === "youtube-nocookie.com" ||
    host === "www.youtube-nocookie.com"
  ) {
    return true;
  }
  if (host.endsWith(".youtube.com") || host.endsWith(".googlevideo.com")) {
    return true;
  }
  return host === "googlevideo.com";
}

/** True when a link should use ODM's YouTube format picker (not plain enqueue). */
function isYouTubePageURL(raw) {
  try {
    const u = new URL(String(raw || "").trim());
    if (u.protocol !== "http:" && u.protocol !== "https:") {
      return false;
    }
    const host = u.hostname.toLowerCase();
    return (
      host === "youtu.be" ||
      host === "youtube.com" ||
      host === "www.youtube.com" ||
      host === "m.youtube.com" ||
      host === "music.youtube.com" ||
      host.endsWith(".youtube.com")
    );
  } catch {
    return false;
  }
}

/** Last path segment of a browser download filename (OS path or bare name). */
function downloadBasename(filename) {
  const raw = String(filename || "").trim();
  if (!raw) {
    return "";
  }
  const normalized = raw.replace(/\\/g, "/");
  const parts = normalized.split("/");
  return parts[parts.length - 1] || "";
}

/**
 * Build a Cookie request header from chrome.cookies.getAll results.
 * Caps total length to match the native host (8 KiB).
 */
function buildCookieHeader(cookies) {
  if (!Array.isArray(cookies) || cookies.length === 0) {
    return "";
  }
  const max = 8 * 1024;
  const parts = [];
  let len = 0;
  for (const c of cookies) {
    if (!c || typeof c.name !== "string") {
      continue;
    }
    const piece = `${c.name}=${c.value == null ? "" : String(c.value)}`;
    const add = (parts.length === 0 ? 0 : 2) + piece.length;
    if (len + add > max) {
      break;
    }
    parts.push(piece);
    len += add;
  }
  return parts.join("; ");
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    shouldInterceptDownload,
    isYouTubeRelatedHost,
    isYouTubePageURL,
    isGitHubNonAssetPage,
    looksLikeOpaqueCDNName,
    urlPathBasename,
    resolveDownloadFilename,
    downloadBasename,
    buildCookieHeader,
  };
}
