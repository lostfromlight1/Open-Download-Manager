/* global shouldInterceptDownload, downloadBasename */
// Chrome MV3 service worker; Firefox loads intercept.js via manifest scripts.
if (typeof importScripts === "function") {
  importScripts("intercept.js");
}

const HOST_NAME = "app.opendownloadmanager.native";
const MENU_ID = "odm-send-link";
const CLICK_SCRIPT_ID = "odm-click-capture";

const interceptInFlight = new Set();

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: MENU_ID,
      title: "Download with ODM",
      contexts: ["link"],
    });
  });
  void chrome.storage.local
    .get({ captureClicks: false, interceptDownloads: false })
    .then((prefs) => {
      void setClickCapture(!!prefs.captureClicks);
      void syncInterceptListener(!!prefs.interceptDownloads);
    });
});

chrome.runtime.onStartup.addListener(() => {
  void chrome.storage.local
    .get({ captureClicks: false, interceptDownloads: false })
    .then((prefs) => {
      void setClickCapture(!!prefs.captureClicks);
      void syncInterceptListener(!!prefs.interceptDownloads);
    });
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
    void (async () => {
      const next = {};
      if ("captureClicks" in msg) {
        next.captureClicks = !!msg.captureClicks;
      }
      if ("interceptDownloads" in msg) {
        next.interceptDownloads = !!msg.interceptDownloads;
      }
      const current = await chrome.storage.local.get({
        captureClicks: false,
        interceptDownloads: false,
      });
      const capture =
        "captureClicks" in next ? next.captureClicks : !!current.captureClicks;
      const intercept =
        "interceptDownloads" in next
          ? next.interceptDownloads
          : !!current.interceptDownloads;

      let ok = true;
      if ("captureClicks" in next) {
        ok = (await setClickCapture(capture)) && ok;
      }
      if ("interceptDownloads" in next) {
        ok = (await setInterceptDownloads(intercept)) && ok;
      }
      const stored = await chrome.storage.local.get({
        captureClicks: false,
        interceptDownloads: false,
      });
      sendResponse({
        ok,
        captureClicks: !!stored.captureClicks,
        interceptDownloads: !!stored.interceptDownloads,
      });
    })();
    return true;
  }
  return false;
});

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
    // not registered yet
  }
  if (!enabled) {
    await chrome.storage.local.set({ captureClicks: false });
    return true;
  }
  // Permission must already have been granted from the popup (user gesture).
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
  // Permission must already have been granted from the popup (user gesture).
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

async function syncInterceptListener(wantEnabled) {
  if (!wantEnabled) {
    removeInterceptListener();
    return;
  }
  const has = await chrome.permissions.contains({
    permissions: ["downloads"],
  });
  if (!has) {
    await chrome.storage.local.set({ interceptDownloads: false });
    removeInterceptListener();
    return;
  }
  attachInterceptListener();
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
  if (!shouldInterceptDownload(item)) {
    return;
  }
  interceptInFlight.add(item.id);
  try {
    const name = downloadBasename(item.filename || "");
    const res = await enqueue(item.url, name);
    if (!res?.ok) {
      return;
    }
    try {
      await chrome.downloads.cancel(item.id);
    } catch {
      // race with browser completion
    }
    try {
      await chrome.downloads.erase({ id: item.id });
    } catch {
      // ignore erase failures
    }
  } finally {
    interceptInFlight.delete(item.id);
  }
}

// Restore listeners after service-worker restart.
void chrome.storage.local
  .get({ interceptDownloads: false })
  .then((prefs) => syncInterceptListener(!!prefs.interceptDownloads));
