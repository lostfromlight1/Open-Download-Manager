// In-page "Download with ODM" on YouTube watch/shorts pages.
(function () {
  const BTN_ID = "odm-yt-download-btn";

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

  function ensureButton() {
    if (!isWatchPage()) {
      const existing = document.getElementById(BTN_ID);
      if (existing) existing.remove();
      return;
    }
    if (document.getElementById(BTN_ID)) {
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
    btn.textContent = "ODM";
    btn.title = "Download with ODM (choose format)…";
    btn.setAttribute("aria-label", "Download with Open Download Manager");
    Object.assign(btn.style, {
      marginLeft: "8px",
      padding: "0 14px",
      height: "36px",
      borderRadius: "18px",
      border: "none",
      cursor: "pointer",
      fontWeight: "600",
      fontSize: "14px",
      background: "#0f766e",
      color: "#fff",
      verticalAlign: "middle",
      flexShrink: "0",
    });

    btn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      btn.disabled = true;
      btn.textContent = "…";
      chrome.runtime.sendMessage(
        { type: "media_pick", url: currentURL() },
        (res) => {
          void chrome.runtime.lastError;
          btn.disabled = false;
          btn.textContent = "ODM";
          if (res && res.ok === false) {
            btn.title =
              res.message ||
              res.error ||
              "ODM rejected the request — check desktop Settings";
          } else {
            btn.title = "Sent to ODM — choose format in the app";
          }
        },
      );
    });

    host.appendChild(btn);
  }

  const obs = new MutationObserver(() => ensureButton());
  obs.observe(document.documentElement, { childList: true, subtree: true });
  ensureButton();
  document.addEventListener("yt-navigate-finish", ensureButton);
})();
