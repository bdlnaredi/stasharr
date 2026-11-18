import React, { useEffect, useRef, useState } from "react";
import { t } from "./i18n";

// A PIN entry with dots and a number pad, for profiles and the discreet
// lock. Typing on the keyboard works too.
export const PinPad: React.FC<{
  title: string;
  subtitle?: string;
  error?: string | null;
  busy?: boolean;
  min?: number;
  onSubmit: (pin: string) => void;
  onCancel?: () => void;
}> = ({ title, subtitle, error, busy, min = 4, onSubmit, onCancel }) => {
  const [pin, setPin] = useState("");
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => {
    input.current?.focus();
  }, []);
  // a wrong PIN clears the dots
  useEffect(() => {
    if (error) setPin("");
  }, [error]);

  const add = (d: string) => setPin((p) => (p.length < 8 ? p + d : p));
  const submit = () => {
    if (pin.length >= min && !busy) onSubmit(pin);
  };

  return (
    <div className="site-pin">
      <h2>{title}</h2>
      {subtitle && <p className="site-pin-sub">{subtitle}</p>}
      <input
        ref={input}
        className="site-pin-input"
        type="password"
        inputMode="numeric"
        autoComplete="off"
        aria-label={title}
        value={pin}
        onChange={(e) => setPin(e.target.value.replace(/\D/g, "").slice(0, 8))}
        onKeyDown={(e) => {
          if (e.key === "Enter") submit();
          if (e.key === "Escape" && onCancel) onCancel();
        }}
      />
      <div className={`site-pin-dots ${error ? "shake" : ""}`} onClick={() => input.current?.focus()}>
        {Array.from({ length: Math.max(4, pin.length) }).map((_, i) => (
          <span key={i} className={i < pin.length ? "on" : ""} />
        ))}
      </div>
      {error && <div className="site-pin-error">{error}</div>}
      <div className="site-pin-pad">
        {["1", "2", "3", "4", "5", "6", "7", "8", "9"].map((d) => (
          <button type="button" key={d} onClick={() => add(d)}>
            {d}
          </button>
        ))}
        <button type="button" className="site-pin-key-muted" onClick={() => setPin((p) => p.slice(0, -1))} aria-label={t("Delete")}>
          ⌫
        </button>
        <button type="button" onClick={() => add("0")}>
          0
        </button>
        <button type="button" className="site-pin-key-go" disabled={pin.length < min || busy} onClick={submit} aria-label={t("OK")}>
          ✓
        </button>
      </div>
      {onCancel && (
        <button type="button" className="site-btn site-btn-ghost site-pin-cancel" onClick={onCancel}>
          {t("Cancel")}
        </button>
      )}
    </div>
  );
};

// SHA-256, written out because the browser's own one only works on https
// and the site is often opened over plain http at home.
export function sha256(text: string): string {
  const K = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5, 0xd807aa98,
    0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786,
    0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152, 0xa831c66d, 0xb00327c8,
    0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819,
    0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a,
    0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7,
    0xc67178f2,
  ];
  const H = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19];
  const bytes = Array.from(new TextEncoder().encode(text));
  const bitLen = bytes.length * 8;
  bytes.push(0x80);
  while (bytes.length % 64 !== 56) bytes.push(0);
  for (let i = 7; i >= 0; i--) bytes.push(i >= 4 ? 0 : (bitLen >>> (i * 8)) & 0xff);
  const rotr = (x: number, n: number) => (x >>> n) | (x << (32 - n));
  const w = new Array<number>(64);
  for (let off = 0; off < bytes.length; off += 64) {
    for (let i = 0; i < 16; i++) {
      w[i] =
        (bytes[off + i * 4] << 24) | (bytes[off + i * 4 + 1] << 16) | (bytes[off + i * 4 + 2] << 8) | bytes[off + i * 4 + 3];
    }
    for (let i = 16; i < 64; i++) {
      const s0 = rotr(w[i - 15], 7) ^ rotr(w[i - 15], 18) ^ (w[i - 15] >>> 3);
      const s1 = rotr(w[i - 2], 17) ^ rotr(w[i - 2], 19) ^ (w[i - 2] >>> 10);
      w[i] = (w[i - 16] + s0 + w[i - 7] + s1) | 0;
    }
    let [a, b, c, d, e, f, g, h] = H;
    for (let i = 0; i < 64; i++) {
      const S1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
      const ch = (e & f) ^ (~e & g);
      const t1 = (h + S1 + ch + K[i] + w[i]) | 0;
      const S0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22);
      const maj = (a & b) ^ (a & c) ^ (b & c);
      const t2 = (S0 + maj) | 0;
      h = g;
      g = f;
      f = e;
      e = (d + t1) | 0;
      d = c;
      c = b;
      b = a;
      a = (t1 + t2) | 0;
    }
    H[0] = (H[0] + a) | 0;
    H[1] = (H[1] + b) | 0;
    H[2] = (H[2] + c) | 0;
    H[3] = (H[3] + d) | 0;
    H[4] = (H[4] + e) | 0;
    H[5] = (H[5] + f) | 0;
    H[6] = (H[6] + g) | 0;
    H[7] = (H[7] + h) | 0;
  }
  return H.map((x) => (x >>> 0).toString(16).padStart(8, "0")).join("");
}
