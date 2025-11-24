import { shareSystemClipboard } from "../api/wailsBridge";
import { EventsOn, WindowGetPosition } from "../../wailsjs/runtime/runtime";

const noteIconUrl = new URL("../../img/note.png", import.meta.url).href;
const SHOW_EVENT = "clipboard:show-share-button";
const PERMISSION_EVENT = "clipboard:permission-state";
const BUTTON_HIDE_DELAY = 4500;

type ClipboardButtonPayload = {
  screenX: number;
  screenY: number;
};

type PermissionPayload = {
  granted: boolean;
  message?: string;
};

let buttonEl: HTMLButtonElement | null = null;
let toastEl: HTMLDivElement | null = null;
let hideTimer: number | null = null;
let toastTimer: number | null = null;
let draggingPointerId: number | null = null;
let dragOffsetX = 0;
let dragOffsetY = 0;
let sharing = false;
let initialized = false;

function ensureButton(): HTMLButtonElement {
  if (buttonEl) {
    return buttonEl;
  }

  buttonEl = document.createElement("button");
  buttonEl.className = "clipboard-floating-button";
  buttonEl.title = "Drop here to share your latest copy";
  buttonEl.setAttribute("aria-label", "Share copied item");
  buttonEl.type = "button";

  const img = document.createElement("img");
  img.src = noteIconUrl;
  img.alt = "Floating clipboard icon";
  buttonEl.appendChild(img);

  buttonEl.addEventListener("pointerdown", (event) => {
    event.preventDefault();
    buttonEl?.setPointerCapture(event.pointerId);
    draggingPointerId = event.pointerId;
    dragOffsetX = event.clientX - buttonEl!.offsetLeft;
    dragOffsetY = event.clientY - buttonEl!.offsetTop;
    buttonEl!.dataset.dragging = "true";
  });

  const stopDragging = (event: PointerEvent, shouldShare: boolean) => {
    if (draggingPointerId !== event.pointerId) {
      return;
    }
    event.preventDefault();
    buttonEl?.releasePointerCapture(event.pointerId);
    draggingPointerId = null;
    buttonEl!.dataset.dragging = "false";
    if (shouldShare) {
      void executeShare();
    }
  };

  buttonEl.addEventListener("pointermove", (event) => {
    if (draggingPointerId !== event.pointerId) {
      return;
    }
    event.preventDefault();
    positionButton(event.clientX - dragOffsetX, event.clientY - dragOffsetY);
  });

  buttonEl.addEventListener("pointerup", (event) => stopDragging(event, true));
  buttonEl.addEventListener("pointercancel", (event) => stopDragging(event, false));

  buttonEl.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      void executeShare();
    }
  });

  document.body.appendChild(buttonEl);
  return buttonEl;
}

function ensureToast(): HTMLDivElement {
  if (toastEl) {
    return toastEl;
  }

  toastEl = document.createElement("div");
  toastEl.className = "clipboard-share-toast";
  document.body.appendChild(toastEl);
  return toastEl;
}

function showToast(message: string, isError = false, duration = 5000): void {
  const el = ensureToast();
  el.textContent = message;
  el.dataset.state = isError ? "error" : "info";
  el.classList.add("visible");

  if (toastTimer) {
    window.clearTimeout(toastTimer);
  }

  toastTimer = window.setTimeout(() => {
    el.classList.remove("visible");
  }, duration);
}

function hideToast(): void {
  if (!toastEl) {
    return;
  }
  if (toastTimer) {
    window.clearTimeout(toastTimer);
    toastTimer = null;
  }
  toastEl.classList.remove("visible");
}

function positionButton(clientX: number, clientY: number): void {
  const button = ensureButton();
  const rect = button.getBoundingClientRect();
  const width = rect.width || 72;
  const height = rect.height || 72;
  const margin = 12;
  const maxX = Math.max(margin, window.innerWidth - width - margin);
  const maxY = Math.max(margin, window.innerHeight - height - margin);
  const clampedX = Math.min(Math.max(clientX, margin), maxX);
  const clampedY = Math.min(Math.max(clientY, margin), maxY);
  button.style.left = `${clampedX}px`;
  button.style.top = `${clampedY}px`;
}

function hideButton(): void {
  const button = ensureButton();
  button.classList.remove("visible");
  button.dataset.dragging = "false";
  draggingPointerId = null;

  if (hideTimer) {
    window.clearTimeout(hideTimer);
    hideTimer = null;
  }
}

async function showButtonNear(payload: ClipboardButtonPayload): Promise<void> {
  const button = ensureButton();
  hideToast();

  let windowPos: { x: number; y: number } | null = null;
  try {
    windowPos = await WindowGetPosition();
  } catch (error) {
    console.warn("Failed to read window position", error);
  }

  const offsetX = windowPos ? payload.screenX - windowPos.x : window.innerWidth / 2;
  const offsetY = windowPos ? payload.screenY - windowPos.y : window.innerHeight / 2;
  positionButton(offsetX, offsetY);

  button.classList.add("visible");

  if (hideTimer) {
    window.clearTimeout(hideTimer);
  }

  hideTimer = window.setTimeout(() => {
    hideButton();
  }, BUTTON_HIDE_DELAY);
}

async function executeShare(): Promise<void> {
  if (sharing) {
    return;
  }

  sharing = true;
  const button = ensureButton();
  button.classList.add("sharing");

  try {
    await shareSystemClipboard();
    showToast("Copied item sent to the room", false, 2500);
  } catch (error) {
    console.error("Failed to share clipboard", error);
    showToast("Couldn't share clipboard. See logs for details.", true);
  } finally {
    sharing = false;
    button.classList.remove("sharing");
    hideButton();
  }
}

function handlePermissionEvent(payload: PermissionPayload): void {
  if (payload.granted) {
    hideToast();
    return;
  }

  const message =
    payload.message ?? "Grant Accessibility permissions in macOS Settings > Privacy & Security > Accessibility.";
  showToast(message, true, 8000);
}

function isClipboardPayload(payload: unknown): payload is ClipboardButtonPayload {
  if (!payload || typeof payload !== "object") {
    return false;
  }
  const candidate = payload as Partial<ClipboardButtonPayload>;
  return typeof candidate.screenX === "number" && typeof candidate.screenY === "number";
}

export function initClipboardShareButton(): void {
  if (initialized) {
    return;
  }
  initialized = true;

  ensureButton();
  EventsOn(SHOW_EVENT, (payload: ClipboardButtonPayload) => {
    if (isClipboardPayload(payload)) {
      void showButtonNear(payload);
    }
  });
  EventsOn(PERMISSION_EVENT, (payload: PermissionPayload) => {
    if (payload) {
      handlePermissionEvent(payload);
    }
  });

  document.addEventListener("visibilitychange", () => {
    if (document.hidden) {
      hideButton();
    }
  });
}
