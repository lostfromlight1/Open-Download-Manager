const statusEl = document.getElementById("status");
const errorEl = document.getElementById("lastError");
const captureEl = document.getElementById("captureClicks");
const interceptEl = document.getElementById("interceptDownloads");
const refreshBtn = document.getElementById("refresh");

async function refresh() {
  errorEl.hidden = true;
  statusEl.textContent = "Checking host…";
  const res = await chrome.runtime.sendMessage({ type: "ping" });
  if (!res?.ok) {
    statusEl.textContent = "Host unavailable";
    errorEl.hidden = false;
    errorEl.textContent =
      res?.message || res?.error || "Cannot reach odm-nmhost";
    return;
  }
  if (!res.enabled) {
    statusEl.textContent =
      "Host reachable — enable integration in ODM Settings";
    return;
  }
  statusEl.textContent = "Connected — browser integration enabled";
}

async function loadPrefs() {
  const prefs = await chrome.runtime.sendMessage({ type: "getPrefs" });
  captureEl.checked = !!prefs?.captureClicks;
  interceptEl.checked = !!prefs?.interceptDownloads;
}

captureEl.addEventListener("change", async () => {
  errorEl.hidden = true;
  const want = captureEl.checked;
  if (want) {
    // Optional host permission must be requested here (user gesture), not in the SW.
    const granted = await chrome.permissions.request({
      origins: ["http://*/*", "https://*/*"],
    });
    if (!granted) {
      captureEl.checked = false;
      errorEl.hidden = false;
      errorEl.textContent =
        "Click capture needs permission to access http(s) pages.";
      return;
    }
  }
  const res = await chrome.runtime.sendMessage({
    type: "setPrefs",
    captureClicks: want,
  });
  captureEl.checked = !!(res?.ok && want);
  if (want && !res?.ok) {
    errorEl.hidden = false;
    errorEl.textContent =
      "Click capture needs permission to access http(s) pages.";
  }
});

interceptEl.addEventListener("change", async () => {
  errorEl.hidden = true;
  const want = interceptEl.checked;
  if (want) {
    // Optional downloads permission must be requested here (user gesture), not in the SW.
    const granted = await chrome.permissions.request({
      permissions: ["downloads"],
    });
    if (!granted) {
      interceptEl.checked = false;
      errorEl.hidden = false;
      errorEl.textContent =
        "Download interception needs the Downloads permission.";
      return;
    }
  } else {
    try {
      await chrome.permissions.remove({ permissions: ["downloads"] });
    } catch {
      // Permission may already be absent.
    }
  }
  const res = await chrome.runtime.sendMessage({
    type: "setPrefs",
    interceptDownloads: want,
  });
  interceptEl.checked = !!(res?.ok && want);
  if (want && !res?.ok) {
    errorEl.hidden = false;
    errorEl.textContent =
      "Could not enable interception (Downloads permission missing).";
  }
});

refreshBtn.addEventListener("click", () => {
  void refresh();
});

void loadPrefs();
void refresh();
