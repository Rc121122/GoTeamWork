export function showError(elementId: string, message: string, duration = 5000): void {
  const element = document.getElementById(elementId);
  if (!element) {
    return;
  }

  element.textContent = message;
  element.style.display = "block";

  window.setTimeout(() => {
    element.style.display = "none";
  }, duration);
}

export function clearChildren(container: HTMLElement): void {
  while (container.firstChild) {
    container.removeChild(container.firstChild);
  }
}
