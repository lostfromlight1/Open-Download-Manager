const statusEl = document.getElementById("status");
const errorEl = document.getElementById("lastError");
const captureEl = document.getElementById("captureClicks");
const interceptEl = document.getElementById("interceptDownloads");
const forwardEl = document.getElementById("forwardCookies");
const refreshBtn = document.getElementById("refresh");

function setStatus(text, state) {
  statusEl.textContent = text;
  statusEl.dataset.state = state;
}

async function refresh() {
  errorEl.hidden = true;
  setStatus("Checking host…", "checking");
  const res = await chrome.runtime.sendMessage({ type: "ping" });
  if (!res?.ok) {
    setStatus("Host unavailable", "error");
    errorEl.hidden = false;
    errorEl.textContent =
      res?.message || res?.error || "Cannot reach odm-nmhost";
    return;
  }
  if (!res.enabled) {
    setStatus("Host reachable — enable integration in ODM Settings", "warn");
    return;
  }
  setStatus("Connected — browser integration enabled", "ok");
}

async function loadPrefs() {
  const prefs = await chrome.runtime.sendMessage({ type: "getPrefs" });
  captureEl.checked = !!prefs?.captureClicks;
  interceptEl.checked = !!prefs?.interceptDownloads;
  forwardEl.checked = !!prefs?.forwardCookies;
}

captureEl.addEventListener("change", async () => {
  errorEl.hidden = true;
  const want = captureEl.checked;
  if (want) {
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
      // already absent
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

forwardEl.addEventListener("change", async () => {
  errorEl.hidden = true;
  const want = forwardEl.checked;
  if (want) {
    const granted = await chrome.permissions.request({
      permissions: ["cookies"],
    });
    if (!granted) {
      forwardEl.checked = false;
      errorEl.hidden = false;
      errorEl.textContent =
        "Cookie handoff needs the Cookies permission.";
      return;
    }
  } else {
    try {
      await chrome.permissions.remove({ permissions: ["cookies"] });
    } catch {
      // already absent
    }
  }
  const res = await chrome.runtime.sendMessage({
    type: "setPrefs",
    forwardCookies: want,
  });
  forwardEl.checked = !!(res?.ok && want);
  if (want && !res?.ok) {
    errorEl.hidden = false;
    errorEl.textContent = "Could not enable cookie handoff.";
  }
});

refreshBtn.addEventListener("click", () => {
  void refresh();
});

void loadPrefs();
void refresh();
