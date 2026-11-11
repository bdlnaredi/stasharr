import React from "react";

// Stasharr: a diagonal ribbon across the top right corner of a scene's
// picture saying it is a trans, lesbian or bisexual scene, in the colours
// of that flag. The server works out which from the performers' genders and
// the tags.

const LABELS: Record<string, string> = {
  trans: "Trans",
  lesbian: "Lesbian",
  bisexual: "Bi",
};

export const SceneRibbon: React.FC<{ badges?: readonly string[] | null }> = ({ badges }) => {
  const list = (badges ?? []).filter((b) => LABELS[b]);
  if (list.length === 0) return null;
  return (
    <div className={`stasharr-ribbon stasharr-ribbon-${list[0]}`} title={list.map((b) => `${LABELS[b]} scene`).join(", ")}>
      <span>{list.map((b) => LABELS[b]).join(" · ")}</span>
    </div>
  );
};

/** The same, as a small pill next to a scene's title. */
export const SceneBadgePills: React.FC<{ badges?: readonly string[] | null }> = ({ badges }) => {
  const list = (badges ?? []).filter((b) => LABELS[b]);
  if (list.length === 0) return null;
  return (
    <span className="stasharr-badge-pills">
      {list.map((b) => (
        <span key={b} className={`stasharr-badge-pill stasharr-ribbon-${b}`}>
          {LABELS[b]}
        </span>
      ))}
    </span>
  );
};
