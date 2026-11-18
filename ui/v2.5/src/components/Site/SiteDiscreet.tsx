// Discreet mode, kept in this browser only: a quick-hide key that jumps to
// a harmless page, a plain tab title and icon, pictures blurred until
// pointed at, and a PIN to get back in.

import React, { useCallback, useEffect, useState } from "react";
import { PinPad, sha256 } from "./SitePinPad";
import { t } from "./i18n";

export interface IDiscreet {
  on: boolean;
  plainTab: boolean;
  blur: boolean;
  // "esc2": Esc twice; "backquote": the ` key
  hideKey: "esc2" | "backquote";
  safeUrl: string;
  // the PIN's hash, or "" for no PIN
  pin: string;
  // lock after the tab has been away this many minutes; 0 = only after hiding
  lockAfter: number;
}

const KEY = "stasharr-discreet";
const LOCKED = "stasharr-discreet-locked";
const AWAY = "stasharr-discreet-away";
const SALT = "stasharr-discreet:";

export const discreetDefaults: IDiscreet = {
  on: false,
  plainTab: true,
  blur: false,
  hideKey: "esc2",
  safeUrl: "https://www.google.com/search?q=weather",
  pin: "",
  lockAfter: 5,
};

export function loadDiscreet(): IDiscreet {
  try {
    const raw = localStorage.getItem(KEY);
    return raw ? { ...discreetDefaults, ...JSON.parse(raw) } : discreetDefaults;
  } catch {
    return discreetDefaults;
  }
}

export function saveDiscreet(d: IDiscreet) {
  try {
    localStorage.setItem(KEY, JSON.stringify(d));
  } catch {
    // private window: not remembered
  }
  window.dispatchEvent(new Event("stasharr-discreet"));
}

export const hashPin = (pin: string) => sha256(SALT + pin);

function flag(key: string, value: string | null) {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    // ignore
  }
}

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

// a grey square, so the tab shows nothing telling
const PLAIN_ICON =
  "data:image/svg+xml," +
  encodeURIComponent("<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16'><rect width='16' height='16' rx='3' fill='#8a8f98'/></svg>");
const PLAIN_TITLE = "Dashboard";

function useSettings(): IDiscreet {
  const [d, setD] = useState(loadDiscreet);
  useEffect(() => {
    const on = () => setD(loadDiscreet());
    window.addEventListener("stasharr-discreet", on);
    window.addEventListener("storage", on);
    return () => {
      window.removeEventListener("stasharr-discreet", on);
      window.removeEventListener("storage", on);
    };
  }, []);
  return d;
}

// Goes to the harmless page at once; back does not come back here.
export function hideNow(d: IDiscreet = loadDiscreet()) {
  document.querySelectorAll("video").forEach((v) => v.pause());
  if (d.pin) flag(LOCKED, "1");
  window.location.replace(d.safeUrl || discreetDefaults.safeUrl);
}

// Runs the discreet mode for the whole site. Renders the lock screen when
// the PIN is needed.
export const Discreet: React.FC = () => {
  const d = useSettings();
  const [locked, setLocked] = useState(() => {
    const s = loadDiscreet();
    if (!s.on || !s.pin) return false;
    if (read(LOCKED) === "1") return true;
    const away = Number(read(AWAY) ?? 0);
    return s.lockAfter > 0 && away > 0 && Date.now() - away > s.lockAfter * 60000;
  });
  const [error, setError] = useState<string | null>(null);

  // the hide key
  useEffect(() => {
    if (!d.on) return;
    let lastEsc = 0;
    const onKey = (e: KeyboardEvent) => {
      if (d.hideKey === "backquote" && e.key === "`") {
        const el = e.target as HTMLElement | null;
        if (el?.closest("input, textarea, [contenteditable]")) return;
        e.preventDefault();
        hideNow(d);
      }
      if (d.hideKey === "esc2" && e.key === "Escape") {
        const now = Date.now();
        if (now - lastEsc < 600) hideNow(d);
        lastEsc = now;
      }
    };
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, [d]);

  // lock after being away
  useEffect(() => {
    if (!d.on || !d.pin) return;
    const onVis = () => {
      if (document.visibilityState === "hidden") {
        flag(AWAY, String(Date.now()));
      } else {
        const away = Number(read(AWAY) ?? 0);
        if (d.lockAfter > 0 && away > 0 && Date.now() - away > d.lockAfter * 60000) setLocked(true);
        flag(AWAY, null);
      }
    };
    document.addEventListener("visibilitychange", onVis);
    window.addEventListener("pagehide", onVis);
    return () => {
      document.removeEventListener("visibilitychange", onVis);
      window.removeEventListener("pagehide", onVis);
    };
  }, [d]);

  // a plain tab title and icon
  useEffect(() => {
    if (!d.on || !d.plainTab) return;
    const icons = Array.from(document.querySelectorAll<HTMLLinkElement>("link[rel~='icon']"));
    const old = icons.map((l) => l.href);
    icons.forEach((l) => (l.href = PLAIN_ICON));
    const oldTitle = document.title;
    const force = () => {
      if (document.title !== PLAIN_TITLE) document.title = PLAIN_TITLE;
    };
    force();
    const head = document.querySelector("head");
    const mo = new MutationObserver(force);
    if (head) mo.observe(head, { subtree: true, childList: true, characterData: true });
    return () => {
      mo.disconnect();
      icons.forEach((l, i) => (l.href = old[i]));
      document.title = oldTitle;
    };
  }, [d.on, d.plainTab]);

  // blurred pictures
  useEffect(() => {
    const on = d.on && d.blur;
    document.body.classList.toggle("site-discreet-blur", on);
    return () => document.body.classList.remove("site-discreet-blur");
  }, [d.on, d.blur]);

  // nothing shows behind the lock, and videos stop
  useEffect(() => {
    if (locked) document.querySelectorAll("video").forEach((v) => v.pause());
    document.body.classList.toggle("site-discreet-locked", locked);
  }, [locked]);

  const unlock = useCallback(
    (pin: string) => {
      if (hashPin(pin) === d.pin) {
        flag(LOCKED, null);
        flag(AWAY, null);
        setError(null);
        setLocked(false);
      } else {
        setError(t("Wrong PIN."));
      }
    },
    [d.pin]
  );

  if (!locked || !d.on || !d.pin) return null;
  return (
    <div className="site-lock" role="dialog" aria-modal="true">
      <PinPad title={t("Enter your PIN")} error={error} onSubmit={unlock} />
    </div>
  );
};

// The settings, on the account page.
export const DiscreetSettings: React.FC = () => {
  const d = useSettings();
  const [settingPin, setSettingPin] = useState(false);
  const set = (patch: Partial<IDiscreet>) => saveDiscreet({ ...d, ...patch });

  return (
    <section className="site-discreet-settings">
      <h2>{t("Discreet mode")}</h2>
      <p className="site-muted">{t("These settings are for this browser only.")}</p>
      <label className="site-switch">
        <input type="checkbox" checked={d.on} onChange={(e) => set({ on: e.target.checked })} />
        <span>{t("Turn on discreet mode")}</span>
      </label>
      <fieldset disabled={!d.on}>
        <label className="site-field">
          <span>{t("Quick-hide key")}</span>
          <select value={d.hideKey} onChange={(e) => set({ hideKey: e.target.value as IDiscreet["hideKey"] })}>
            <option value="esc2">{t("Press Esc twice")}</option>
            <option value="backquote">{t("Press the ` key")}</option>
          </select>
        </label>
        <label className="site-field">
          <span>{t("Page to jump to")}</span>
          <input
            type="url"
            value={d.safeUrl}
            onChange={(e) => set({ safeUrl: e.target.value })}
            placeholder={discreetDefaults.safeUrl}
          />
        </label>
        <label className="site-switch">
          <input type="checkbox" checked={d.plainTab} onChange={(e) => set({ plainTab: e.target.checked })} />
          <span>{t("Plain tab title and icon")}</span>
        </label>
        <label className="site-switch">
          <input type="checkbox" checked={d.blur} onChange={(e) => set({ blur: e.target.checked })} />
          <span>{t("Blur pictures until pointed at")}</span>
        </label>
        <div className="site-field">
          <span>{t("PIN to get back in")}</span>
          {d.pin ? (
            <div className="site-discreet-pin">
              <span>{t("PIN is on")}</span>
              <button type="button" className="site-btn site-btn-ghost" onClick={() => setSettingPin(true)}>
                {t("Change")}
              </button>
              <button type="button" className="site-btn site-btn-ghost" onClick={() => set({ pin: "" })}>
                {t("Remove")}
              </button>
            </div>
          ) : (
            <button type="button" className="site-btn site-btn-ghost" onClick={() => setSettingPin(true)}>
              {t("Set a PIN")}
            </button>
          )}
        </div>
        {d.pin && (
          <label className="site-field">
            <span>{t("Lock when I come back after")}</span>
            <select value={d.lockAfter} onChange={(e) => set({ lockAfter: Number(e.target.value) })}>
              <option value={0}>{t("Only after quick-hide")}</option>
              <option value={1}>{t("1 minute")}</option>
              <option value={5}>{t("5 minutes")}</option>
              <option value={15}>{t("15 minutes")}</option>
              <option value={60}>{t("1 hour")}</option>
            </select>
          </label>
        )}
        <button type="button" className="site-btn site-btn-primary" onClick={() => hideNow(d)}>
          {t("Try it: hide now")}
        </button>
      </fieldset>
      {settingPin && (
        <div className="site-lock" role="dialog" aria-modal="true">
          <PinPad
            title={t("Choose a PIN")}
            subtitle={t("4 to 8 digits. If you forget it, clear this site's data in the browser.")}
            onSubmit={(pin) => {
              set({ pin: hashPin(pin) });
              setSettingPin(false);
            }}
            onCancel={() => setSettingPin(false)}
          />
        </div>
      )}
    </section>
  );
};
