/** Request permission and show an opt-in completion notification when allowed. */
export async function notifyDownloadComplete(fileName: string): Promise<void> {
  if (typeof Notification === "undefined") return;
  let permission = Notification.permission;
  if (permission === "default") {
    permission = await Notification.requestPermission();
  }
  if (permission !== "granted") return;
  try {
    new Notification("Download complete", {
      body: fileName,
      silent: false,
    });
  } catch {
    // WebView may reject Notification construction; ignore quietly.
  }
}
