// Profiles: "Who's watching?", switching with a PIN, and managing them.

import React, { useState } from "react";
import { createPortal } from "react-dom";
import { useHistory } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { baseURL } from "src/core/createClient";
import { useCurrentUser } from "src/components/Stasharr/CurrentUser";
import { PinPad } from "./SitePinPad";
import { t } from "./i18n";

export type SiteProfile = GQL.SiteProfileDataFragment;

export const profileColors = ["#e84a7f", "#7c5cff", "#16a3b8", "#f2a63b", "#3fbf6f", "#ef5a3c", "#5b7cfa", "#a36bd6"];

const PICKED = "stasharr-profile-picked";

export function profileColor(p: { id: string; color: string }): string {
  if (p.color) return p.color;
  return profileColors[Number(p.id) % profileColors.length] ?? profileColors[0];
}

// Asked once per browser session when the account has profiles.
export function profileWasPicked(): boolean {
  try {
    return sessionStorage.getItem(PICKED) === "1";
  } catch {
    return true;
  }
}

function markPicked() {
  try {
    sessionStorage.setItem(PICKED, "1");
  } catch {
    // ignore
  }
}

// Moves the session to the profile, then reloads the site as that profile.
export async function switchProfile(id: string, pin: string): Promise<string | null> {
  try {
    const res = await fetch(`${baseURL}profiles/switch`, {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id, pin }),
    });
    const body = (await res.json().catch(() => ({}))) as { ok?: boolean; error?: string };
    if (!res.ok || !body.ok) return body.error || t("Could not switch profile.");
    markPicked();
    window.location.href = `${baseURL}site`;
    return null;
  } catch {
    return t("Could not switch profile.");
  }
}

export const ProfileAvatar: React.FC<{ profile: { id: string; name: string; color: string }; size?: number }> = ({
  profile,
  size = 40,
}) => (
  <span
    className="site-profile-avatar"
    style={{ background: profileColor(profile), width: size, height: size, fontSize: size * 0.42 }}
    aria-hidden="true"
  >
    {profile.name.slice(0, 1).toUpperCase()}
  </span>
);

// Asks for the PIN when the profile has one, then switches.
export function useProfileSwitch() {
  const [asking, setAsking] = useState<SiteProfile | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const go = async (p: SiteProfile, pin = "") => {
    if (p.is_current) {
      markPicked();
      window.location.href = `${baseURL}site`;
      return;
    }
    if (p.has_pin && !pin) {
      setError(null);
      setAsking(p);
      return;
    }
    setBusy(true);
    const err = await switchProfile(p.id, pin);
    setBusy(false);
    if (err) setError(err);
  };

  // drawn at the top of the site, so the header's blur cannot trap it
  const host = typeof document !== "undefined" ? document.querySelector(".site-root") ?? document.body : null;
  const dialog =
    asking &&
    host &&
    createPortal(
    <div className="site-lock" role="dialog" aria-modal="true">
      <PinPad
        title={t("Enter the PIN for {name}", { name: asking.name })}
        error={error}
        busy={busy}
        onSubmit={(pin) => go(asking, pin)}
        onCancel={() => setAsking(null)}
      />
    </div>,
      host
    );
  return { go, dialog, error };
}

const ProfileEditor: React.FC<{
  profile?: SiteProfile;
  onDone: () => void;
}> = ({ profile, onDone }) => {
  const [name, setName] = useState(profile?.name ?? "");
  const [color, setColor] = useState(profile?.color || profileColors[0]);
  const [error, setError] = useState<string | null>(null);
  const [settingPin, setSettingPin] = useState(false);
  const refetch = { refetchQueries: ["SiteProfiles"] };
  const [create] = GQL.useSiteProfileCreateMutation(refetch);
  const [update] = GQL.useSiteProfileUpdateMutation(refetch);
  const [remove] = GQL.useSiteProfileDeleteMutation(refetch);
  const [setPin] = GQL.useSiteProfilePinMutation(refetch);
  const isMain = profile?.is_main ?? false;

  const save = async () => {
    try {
      if (!profile) await create({ variables: { name, color } });
      else if (!isMain) await update({ variables: { id: profile.id, name, color } });
      onDone();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <div className="site-profile-editor">
      <h2>{profile ? t("Edit profile") : t("Add a profile")}</h2>
      {!isMain && (
        <>
          <ProfileAvatar profile={{ id: profile?.id ?? "0", name: name || "?", color }} size={96} />
          <label className="site-field">
            <span>{t("Name")}</span>
            <input value={name} maxLength={32} autoFocus onChange={(e) => setName(e.target.value)} />
          </label>
          <div className="site-profile-colors" role="radiogroup" aria-label={t("Colour")}>
            {profileColors.map((c) => (
              <button
                type="button"
                key={c}
                role="radio"
                aria-checked={c === color}
                className={c === color ? "on" : ""}
                style={{ background: c }}
                onClick={() => setColor(c)}
              />
            ))}
          </div>
        </>
      )}
      {profile && (
        <div className="site-field">
          <span>{t("PIN")}</span>
          <div className="site-discreet-pin">
            <span>{profile.has_pin ? t("PIN is on") : t("No PIN")}</span>
            <button type="button" className="site-btn site-btn-ghost" onClick={() => setSettingPin(true)}>
              {profile.has_pin ? t("Change") : t("Set a PIN")}
            </button>
            {profile.has_pin && (
              <button
                type="button"
                className="site-btn site-btn-ghost"
                onClick={() => setPin({ variables: { id: profile.id, pin: "" } }).catch((e) => setError(e.message))}
              >
                {t("Remove")}
              </button>
            )}
          </div>
          {isMain && !profile.has_pin && (
            <small className="site-muted">
              {t("Without a PIN, anyone on a profile can switch to this main profile.")}
            </small>
          )}
        </div>
      )}
      {error && <div className="site-pin-error">{error}</div>}
      <div className="site-profile-editor-actions">
        {!isMain && (
          <button type="button" className="site-btn site-btn-primary" disabled={!name.trim()} onClick={save}>
            {t("Save")}
          </button>
        )}
        <button type="button" className="site-btn site-btn-ghost" onClick={onDone}>
          {isMain ? t("Done") : t("Cancel")}
        </button>
        {profile && !isMain && (
          <button
            type="button"
            className="site-btn site-btn-ghost site-danger"
            onClick={async () => {
              if (!window.confirm(t("Delete {name}? Its history, likes and lists go too.", { name: profile.name }))) return;
              try {
                await remove({ variables: { id: profile.id } });
                onDone();
              } catch (e) {
                setError((e as Error).message);
              }
            }}
          >
            {t("Delete profile")}
          </button>
        )}
      </div>
      {settingPin && profile && (
        <div className="site-lock" role="dialog" aria-modal="true">
          <PinPad
            title={t("Choose a PIN for {name}", { name: profile.name })}
            subtitle={t("4 to 8 digits.")}
            onSubmit={async (pin) => {
              try {
                await setPin({ variables: { id: profile.id, pin } });
                setSettingPin(false);
              } catch (e) {
                setError((e as Error).message);
                setSettingPin(false);
              }
            }}
            onCancel={() => setSettingPin(false)}
          />
        </div>
      )}
    </div>
  );
};

// "Who's watching?"
export const ProfilesPage: React.FC = () => {
  const history = useHistory();
  const { me } = useCurrentUser();
  const { data, loading } = GQL.useSiteProfilesQuery({ fetchPolicy: "cache-and-network" });
  const [manage, setManage] = useState(false);
  const [editing, setEditing] = useState<SiteProfile | "new" | null>(null);
  const { go, dialog, error } = useProfileSwitch();
  const profiles = data?.siteProfiles ?? [];
  const onMain = !me?.is_profile;

  if (loading && profiles.length === 0) return <div className="site-profiles" />;
  if (!me?.multi_user_enabled) {
    return (
      <div className="site-profiles">
        <h1>{t("Profiles")}</h1>
        <p className="site-muted">{t("Profiles need sign in to be turned on in Settings → Security.")}</p>
      </div>
    );
  }

  if (editing) {
    return (
      <div className="site-profiles">
        <ProfileEditor profile={editing === "new" ? undefined : editing} onDone={() => setEditing(null)} />
      </div>
    );
  }

  return (
    <div className="site-profiles">
      <h1>{manage ? t("Manage profiles") : t("Who's watching?")}</h1>
      <div className="site-profile-grid">
        {profiles.map((p) => (
          <button
            type="button"
            key={p.id}
            className={`site-profile-tile ${p.is_current ? "current" : ""} ${manage ? "manage" : ""}`}
            onClick={() => (manage ? setEditing(p) : go(p))}
          >
            <span className="site-profile-face">
              <ProfileAvatar profile={p} size={120} />
              {manage && <span className="site-profile-edit">✎</span>}
              {!manage && p.has_pin && <span className="site-profile-lock">🔒</span>}
            </span>
            <span className="site-profile-name">{p.name}</span>
            {p.is_main && <small>{t("Main profile")}</small>}
          </button>
        ))}
        {manage && onMain && profiles.length < 6 && (
          <button type="button" className="site-profile-tile site-profile-add" onClick={() => setEditing("new")}>
            <span className="site-profile-face">
              <span className="site-profile-avatar site-profile-plus">+</span>
            </span>
            <span className="site-profile-name">{t("Add profile")}</span>
          </button>
        )}
      </div>
      {error && <div className="site-pin-error">{error}</div>}
      {onMain ? (
        <button type="button" className="site-btn site-btn-ghost site-profiles-manage" onClick={() => setManage(!manage)}>
          {manage ? t("Done") : t("Manage profiles")}
        </button>
      ) : (
        <p className="site-muted">{t("Switch to the main profile to add or change profiles.")}</p>
      )}
      {!manage && (
        <button type="button" className="site-link-button" onClick={() => history.push("/site")}>
          {t("Back to the site")}
        </button>
      )}
      {dialog}
    </div>
  );
};
