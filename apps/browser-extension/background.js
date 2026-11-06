/* global shouldInterceptDownload, downloadBasename */
// Chrome MV3 service worker loads helpers via importScripts.
// Firefox lists intercept.js before this file in manifest.firefox.json.
if (typeof importScripts === "function") {
  importScripts("intercept.js");
}

const HOST_NAME = "app.opendownloadmanager.native";
const MENU_ID = "odm-send-link";
const CLICK_SCRIPT_ID = "odm-click-capture";

/** Download IDs currently being handed off to ODM (avoid double cancel). */
const interceptInFlight = new Set();

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: MENU_ID,
      title: "Download with ODM",
      contexts: ["link"],
    });
  });
  void restorePrefs();
});

chrome.runtime.onStartup.addListener(() => {
  void restorePrefs();
});

chrome.contextMenus.onClicked.addListener((info) => {
  if (info.menuItemId !== MENU_ID || !info.linkUrl) {
    return;
  }
  void enqueue(info.linkUrl, "").then((result) => {
    void chrome.storage.session?.set?.({ lastResult: result });
  });
});

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg?.type === "ping") {
    void ping().then(sendResponse);
    return true;
  }
  if (msg?.type === "enqueue" && typeof msg.url === "string") {
    void enqueue(msg.url, msg.filename || "").then(sendResponse);
    return true;
  }
  if (msg?.type === "getPrefs") {
    void chrome.storage.local
      .get({ captureClicks: false, interceptDownloads: false })
      .then(sendResponse);
    return true;
  }
  if (msg?.type === "setPrefs") {
    void applyPrefs(msg).then(sendResponse);
    return true;
  }
  return false;
});

/** Re-apply stored prefs after install, browser startup, or SW wake. */
async function restorePrefs() {
  const prefs = await chrome.storage.local.get({
    captureClicks: false,
    interceptDownloads: false,
  });
  await setClickCapture(!!prefs.captureClicks);
  await setInterceptDownloads(!!prefs.interceptDownloads);
}

/**
 * Apply popup preference changes. Does not call permissions.request —
 * the popup must already have obtained optional permissions under a user gesture.
 */
async function applyPrefs(msg) {
  let ok = true;
  // Only touch keys the popup sent so one toggle cannot disturb the other.
  if ("captureClicks" in msg) {
    ok = (await setClickCapture(!!msg.captureClicks)) && ok;
  }
  if ("interceptDownloads" in msg) {
    ok = (await setInterceptDownloads(!!msg.interceptDownloads)) && ok;
  }

  const stored = await chrome.storage.local.get({
    captureClicks: false,
    interceptDownloads: false,
  });
  return {
    ok,
    captureClicks: !!stored.captureClicks,
    interceptDownloads: !!stored.interceptDownloads,
  };
}

function sendNative(payload) {
  return new Promise((resolve) => {
    try {
      chrome.runtime.sendNativeMessage(HOST_NAME, payload, (response) => {
        const err = chrome.runtime.lastError;
        if (err) {
          resolve({ ok: false, error: "native_error", message: err.message });
          return;
        }
        resolve(
          response || {
            ok: false,
            error: "empty",
            message: "empty host response",
          },
        );
      });
    } catch (e) {
      resolve({
        ok: false,
        error: "native_error",
        message: e instanceof Error ? e.message : String(e),
      });
    }
  });
}

function ping() {
  return sendNative({ v: 1, action: "ping" });
}

function enqueue(url, filename) {
  return sendNative({ v: 1, action: "enqueue", url, filename });
}

async function setClickCapture(enabled) {
  if (!chrome.scripting?.registerContentScripts) {
    return false;
  }
  try {
    await chrome.scripting.unregisterContentScripts({
      ids: [CLICK_SCRIPT_ID],
    });
  } catch {
    // Script was not registered yet.
  }
  if (!enabled) {
    await chrome.storage.local.set({ captureClicks: false });
    return true;
  }
  // Host permission is requested in popup.js during the checkbox click.
  const has = await chrome.permissions.contains({
    origins: ["http://*/*", "https://*/*"],
  });
  if (!has) {
    await chrome.storage.local.set({ captureClicks: false });
    return false;
  }
  await chrome.scripting.registerContentScripts([
    {
      id: CLICK_SCRIPT_ID,
      js: ["content-click.js"],
      matches: ["http://*/*", "https://*/*"],
      runAt: "document_idle",
      persistAcrossSessions: true,
    },
  ]);
  await chrome.storage.local.set({ captureClicks: true });
  return true;
}

async function setInterceptDownloads(enabled) {
  if (!enabled) {
    await chrome.storage.local.set({ interceptDownloads: false });
    removeInterceptListener();
    return true;
  }
  // downloads permission is requested in popup.js during the checkbox click.
  const has = await chrome.permissions.contains({
    permissions: ["downloads"],
  });
  if (!has) {
    await chrome.storage.local.set({ interceptDownloads: false });
    removeInterceptListener();
    return false;
  }
  await chrome.storage.local.set({ interceptDownloads: true });
  attachInterceptListener();
  return true;
}

function attachInterceptListener() {
  if (!chrome.downloads?.onCreated) {
    return;
  }
  if (chrome.downloads.onCreated.hasListener(onDownloadCreated)) {
    return;
  }
  chrome.downloads.onCreated.addListener(onDownloadCreated);
}

function removeInterceptListener() {
  if (!chrome.downloads?.onCreated) {
    return;
  }
  if (chrome.downloads.onCreated.hasListener(onDownloadCreated)) {
    chrome.downloads.onCreated.removeListener(onDownloadCreated);
  }
}

function onDownloadCreated(item) {
  void handleDownloadCreated(item);
}

async function handleDownloadCreated(item) {
  if (!item || interceptInFlight.has(item.id)) {
    return;
  }
  const prefs = await chrome.storage.local.get({ interceptDownloads: false });
  if (!prefs.interceptDownloads) {
    return;
  }

  // Prefer finalUrl after redirects; fall back to the original request URL.
  const url = String(item.finalUrl || item.url || "").trim();
  const candidate = { ...item, url };
  if (!shouldInterceptDownload(candidate)) {
    return;
  }

  interceptInFlight.add(item.id);
  try {
    const name = downloadBasename(item.filename || "");
    const res = await enqueue(url, name);
    // Leave the browser download alone if ODM rejected or is unreachable.
    if (!res?.ok) {
      return;
    }
    try {
      await chrome.downloads.cancel(item.id);
    } catch {
      // Browser may have finished before cancel.
    }
    try {
      await chrome.downloads.erase({ id: item.id });
    } catch {
      // Erase is best-effort cleanup of the cancelled item.
    }
  } finally {
    interceptInFlight.delete(item.id);
  }
}

// Service workers can restart while the browser stays open; re-bind listeners.
void restorePrefs();
