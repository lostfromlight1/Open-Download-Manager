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
  let url;
  try {
    url = new URL(item.url);
  } catch {
    return false;
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") {
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

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    shouldInterceptDownload,
    downloadBasename,
  };
}
