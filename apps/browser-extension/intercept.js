/**
 * Pure helpers for opt-in browser download interception.
 * Loaded via importScripts (Chrome SW) or script order (Firefox).
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
  if (
    mime.startsWith("text/html") ||
    mime.startsWith("application/xhtml")
  ) {
    return false;
  }
  return true;
}

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
