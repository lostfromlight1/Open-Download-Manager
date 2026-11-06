// Optional: only injected when the user enables click capture in the popup.
(function () {
  if (window.__odmClickCapture) return;
  window.__odmClickCapture = true;

  document.addEventListener(
    "click",
    (event) => {
      if (event.defaultPrevented || event.button !== 0) return;
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      const a = event.target && event.target.closest ? event.target.closest("a[href]") : null;
      if (!a) return;
      const href = a.href || "";
      if (!/^https?:\/\//i.test(href)) return;
      event.preventDefault();
      event.stopPropagation();
      chrome.runtime.sendMessage({ type: "enqueue", url: href }, () => {
        void chrome.runtime.lastError;
      });
    },
    true,
  );
})();
