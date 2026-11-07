// Optional: only injected when the user enables click capture in the popup.
(function () {
  if (window.__odmClickCapture) return;
  window.__odmClickCapture = true;

  function isYouTubeHost(hostname) {
    const host = String(hostname || "")
      .toLowerCase()
      .replace(/\.$/, "");
    return (
      host === "youtu.be" ||
      host === "youtube.com" ||
      host === "www.youtube.com" ||
      host === "m.youtube.com" ||
      host === "music.youtube.com" ||
      host.endsWith(".youtube.com")
    );
  }

  // Never steal clicks on YouTube itself — that blocks watching videos.
  if (isYouTubeHost(location.hostname)) {
    return;
  }

  document.addEventListener(
    "click",
    (event) => {
      if (event.defaultPrevented || event.button !== 0) return;
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      const a = event.target && event.target.closest ? event.target.closest("a[href]") : null;
      if (!a) return;
      const href = a.href || "";
      if (!/^https?:\/\//i.test(href)) return;
      try {
        if (isYouTubeHost(new URL(href).hostname)) {
          // YouTube links use the context-menu format picker, not click capture.
          return;
        }
      } catch {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      chrome.runtime.sendMessage({ type: "enqueue", url: href }, () => {
        void chrome.runtime.lastError;
      });
    },
    true,
  );
})();
