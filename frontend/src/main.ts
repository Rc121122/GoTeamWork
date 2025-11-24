import "./style.css";
import "./app.css";

import { getAppMode } from "./api/wailsBridge";
import { renderHostLobby } from "./host";
import { renderClientUI } from "./client";
import { initClipboardShareButton } from "./ui/hotkeyIndicator";

async function initApp(): Promise<void> {
  try {
    const mode = await getAppMode();

    if (mode === "host") {
      renderHostLobby();
    } else {
      renderClientUI();
    }
  } catch (error) {
    console.error("Failed to determine app mode", error);
    renderClientUI();
  }
}

document.addEventListener("DOMContentLoaded", () => {
  initClipboardShareButton();
  void initApp();
});
