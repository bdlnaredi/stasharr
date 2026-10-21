import React from "react";

// A settings row in the same layout as the upstream settings pages.
export const SettingRow: React.FC<{
  heading: React.ReactNode;
  sub?: React.ReactNode;
}> = ({ heading, sub, children }) => (
  <div className="setting">
    <div>
      <h3>{heading}</h3>
      {sub && <div className="sub-heading">{sub}</div>}
    </div>
    <div>{children}</div>
  </div>
);

const sizeUnits = ["B", "KB", "MB", "GB", "TB", "PB"];

export function formatBytes(bytes: number | null | undefined): string {
  let n = bytes ?? 0;
  let u = 0;
  while (n >= 1024 && u < sizeUnits.length - 1) {
    n /= 1024;
    u++;
  }
  const digits = u >= 3 ? 1 : 0;
  return `${n.toFixed(digits)} ${sizeUnits[u]}`;
}

export function baseName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

export function dirName(path: string): string {
  const parts = path.split(/[\\/]/);
  parts.pop();
  return parts.join(path.includes("\\") && !path.includes("/") ? "\\" : "/");
}

export function resolutionLabel(width: number, height: number): string {
  if (!width || !height) return "";
  const short = Math.min(width, height);
  return `${short}p`;
}

export function formatBitrate(bitsPerSecond: number): string {
  if (!bitsPerSecond) return "";
  return `${(bitsPerSecond / 1000000).toFixed(1)} Mb/s`;
}

export function formatDuration(seconds: number | null | undefined): string {
  const s = Math.round(seconds ?? 0);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const mm = h > 0 ? String(m).padStart(2, "0") : String(m);
  return `${h > 0 ? `${h}:` : ""}${mm}:${String(sec).padStart(2, "0")}`;
}

export function formatDate(value: string | null | undefined): string {
  if (!value) return "";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleDateString();
}

export function sceneName(scene: {
  title?: string | null;
  files: { path: string }[];
}): string {
  if (scene.title) return scene.title;
  const path = scene.files[0]?.path ?? "";
  return baseName(path);
}
