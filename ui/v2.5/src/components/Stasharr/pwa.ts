// Installable app support: registers the service worker and keeps the
// browser's install prompt so the Account page can offer an Install button.

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

let deferredPrompt: BeforeInstallPromptEvent | undefined;
const listeners = new Set<() => void>();

function notify() {
  listeners.forEach((l) => l());
}

export function registerStasharrApp() {
  window.addEventListener("beforeinstallprompt", (e) => {
    e.preventDefault();
    deferredPrompt = e as BeforeInstallPromptEvent;
    notify();
  });
  window.addEventListener("appinstalled", () => {
    deferredPrompt = undefined;
    notify();
  });

  if (!("serviceWorker" in navigator) || !window.isSecureContext) return;
  window.addEventListener("load", () => {
    // relative to the page's base, so it works behind a path prefix
    navigator.serviceWorker.register("sw.js", { scope: "./" }).catch((err) => {
      console.warn("Could not register the Stasharr service worker", err);
    });
  });
}

export function canInstall(): boolean {
  return !!deferredPrompt;
}

export function isInstalled(): boolean {
  return (
    window.matchMedia?.("(display-mode: standalone)").matches ||
    (navigator as unknown as { standalone?: boolean }).standalone === true
  );
}

export async function promptInstall(): Promise<boolean> {
  if (!deferredPrompt) return false;
  const p = deferredPrompt;
  deferredPrompt = undefined;
  await p.prompt();
  const choice = await p.userChoice;
  notify();
  return choice.outcome === "accepted";
}

export function onInstallChange(fn: () => void): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
