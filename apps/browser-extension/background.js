const HOST_NAME = "app.opendownloadmanager.native";
const MENU_ID = "odm-send-link";
const CLICK_SCRIPT_ID = "odm-click-capture";

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: MENU_ID,
      title: "Download with ODM",
      contexts: ["link"],
    });
  });
  void chrome.storage.local.get({ captureClicks: false }).then((prefs) => {
    void setClickCapture(!!prefs.captureClicks);
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
    void chrome.storage.local.get({ captureClicks: false }).then(sendResponse);
    return true;
  }
  if (msg?.type === "setPrefs") {
    const enabled = !!msg.captureClicks;
    void chrome.storage.local.set({ captureClicks: enabled }).then(async () => {
      const ok = await setClickCapture(enabled);
      sendResponse({ ok });
    });
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
    return true;
  }
  const granted = await chrome.permissions.request({
    origins: ["http://*/*", "https://*/*"],
  });
  if (!granted) {
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
  return true;
}
