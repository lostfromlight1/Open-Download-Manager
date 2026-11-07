// In-page "Download with ODM" on YouTube watch/shorts pages.
(function () {
  const BTN_ID = "odm-yt-download-btn";
  const LABEL = "↓ ODM";
  const RELOAD_MSG =
    "Extension was reloaded — refresh this YouTube tab, then try again";

  function isWatchPage() {
    const path = location.pathname || "";
    return (
      path === "/watch" ||
      path.startsWith("/shorts/") ||
      path.startsWith("/live/")
    );
  }

  function currentURL() {
    return location.href.split("#")[0];
  }

  /** False after the extension is reloaded/disabled while this tab stays open. */
  function extensionAlive() {
    try {
      return Boolean(chrome.runtime && chrome.runtime.id);
    } catch {
      return false;
    }
  }

  function runtimeErrorMessage() {
    try {
      return chrome.runtime.lastError?.message || "";
    } catch {
      return "Extension context invalidated.";
    }
  }

  function isContextDeadError(msg) {
    return /extension context invalidated|context invalidated|receiving end does not exist/i.test(
      String(msg || ""),
    );
  }

  function setButtonState(btn, { text, title, background, disabled }) {
    if (!btn || document.getElementById(BTN_ID) !== btn) return;
    if (text != null) btn.textContent = text;
    if (title != null) btn.title = title;
    if (background != null) btn.style.background = background;
    if (disabled != null) {
      btn.disabled = disabled;
      btn.style.opacity = disabled ? "0.75" : "1";
    }
  }

  function showError(btn, message) {
    setButtonState(btn, {
      text: "Retry",
      title: message,
      background: "#b45309",
      disabled: false,
    });
  }

  function showSent(btn) {
    setButtonState(btn, {
      text: "Sent ✓",
      title: "Sent to ODM — choose format in the app",
      background: "#166534",
      disabled: false,
    });
    setTimeout(() => {
      setButtonState(btn, {
        text: LABEL,
        title: "Download with ODM (choose format)…",
        background: "#0f766e",
      });
    }, 1800);
  }

  /**
   * Ask the service worker to open the desktop format picker.
   * Never throws — invalidated extension contexts resolve to a reload hint.
   */
  function requestMediaPick(url) {
    return new Promise((resolve) => {
      if (!extensionAlive()) {
        resolve({ ok: false, message: RELOAD_MSG });
        return;
      }
      try {
        chrome.runtime.sendMessage({ type: "media_pick", url }, (res) => {
          const err = runtimeErrorMessage();
          if (err || !extensionAlive()) {
            resolve({
              ok: false,
              message: isContextDeadError(err) ? RELOAD_MSG : err || RELOAD_MSG,
            });
            return;
          }
          if (res && res.ok === false) {
            resolve({
              ok: false,
              message:
                res.message ||
                res.error ||
                "ODM rejected the request — check desktop Settings",
            });
            return;
          }
          resolve({ ok: true });
        });
      } catch {
        resolve({ ok: false, message: RELOAD_MSG });
      }
    });
  }

  function ensureButton() {
    if (!isWatchPage()) {
      const existing = document.getElementById(BTN_ID);
      if (existing) existing.remove();
      return;
    }
    if (document.getElementById(BTN_ID)) {
      return;
    }
    // Do not inject a dead button after the extension was reloaded.
    if (!extensionAlive()) {
      return;
    }

    const host =
      document.querySelector("#actions #top-level-buttons-computed") ||
      document.querySelector("#actions #menu") ||
      document.querySelector("ytd-watch-metadata #actions") ||
      document.querySelector("#above-the-fold #actions") ||
      document.querySelector("#owner");

    if (!host) {
      return;
    }

    const btn = document.createElement("button");
    btn.id = BTN_ID;
    btn.type = "button";
    btn.textContent = LABEL;
    btn.title = "Download with ODM (choose format)…";
    btn.setAttribute("aria-label", "Download with Open Download Manager");
    Object.assign(btn.style, {
      marginLeft: "8px",
      padding: "0 16px",
      height: "36px",
      borderRadius: "18px",
      border: "none",
      cursor: "pointer",
      fontWeight: "650",
      fontSize: "13px",
      letterSpacing: "0.02em",
      background: "#0f766e",
      color: "#fff",
      verticalAlign: "middle",
      flexShrink: "0",
      boxShadow: "0 1px 2px rgba(15, 118, 110, 0.35)",
      transition: "background 120ms ease, opacity 120ms ease",
    });

    btn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (!extensionAlive()) {
        showError(btn, RELOAD_MSG);
        return;
      }
      setButtonState(btn, {
        text: "Sending…",
        disabled: true,
        background: "#0f766e",
      });
      void requestMediaPick(currentURL()).then((res) => {
        if (res.ok) {
          showSent(btn);
        } else {
          showError(btn, res.message || RELOAD_MSG);
        }
      });
    });

    host.appendChild(btn);
  }

  const obs = new MutationObserver(() => {
    if (!extensionAlive()) {
      try {
        obs.disconnect();
      } catch {
        // ignore
      }
      const existing = document.getElementById(BTN_ID);
      if (existing) {
        existing.title = RELOAD_MSG;
        existing.textContent = "Reload tab";
        existing.style.background = "#b45309";
        existing.disabled = true;
      }
      return;
    }
    ensureButton();
  });
  obs.observe(document.documentElement, { childList: true, subtree: true });
  ensureButton();
  document.addEventListener("yt-navigate-finish", ensureButton);
})();
