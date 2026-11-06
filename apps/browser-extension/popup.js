const statusEl = document.getElementById("status");
const errorEl = document.getElementById("lastError");
const captureEl = document.getElementById("captureClicks");
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
}

captureEl.addEventListener("change", async () => {
  const res = await chrome.runtime.sendMessage({
    type: "setPrefs",
    captureClicks: captureEl.checked,
  });
  if (captureEl.checked && !res?.ok) {
    captureEl.checked = false;
    errorEl.hidden = false;
    errorEl.textContent =
      "Click capture needs permission to access http(s) pages.";
  }
});

refreshBtn.addEventListener("click", () => {
  void refresh();
});

void loadPrefs();
void refresh();
