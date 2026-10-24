import React, { useEffect, useState } from "react";
import { Button, Form } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { baseURL } from "src/core/createClient";
import { useCurrentUser } from "./CurrentUser";
import { canInstall, isInstalled, onInstallChange, promptInstall } from "./pwa";

// Install Stasharr as an app on this phone or computer.
export const InstallAppSection: React.FC = () => {
  const [, setTick] = useState(0);
  useEffect(() => onInstallChange(() => setTick((t) => t + 1)), []);

  let sub: React.ReactNode;
  if (isInstalled()) {
    sub = "Stasharr is running as an installed app.";
  } else if (!window.isSecureContext) {
    sub = (
      <>
        Phones only install apps from secure (https) addresses. Open Stasharr
        through https, for example with Tailscale or a reverse proxy, then come
        back here. Until then you can still use your browser&apos;s
        &quot;Add to Home screen&quot;.
      </>
    );
  } else if (canInstall()) {
    sub = "Adds Stasharr to your home screen. It opens full screen, without the browser bars.";
  } else {
    sub = (
      <>
        Open your browser menu and choose &quot;Install app&quot; or
        &quot;Add to Home screen&quot;. On iPhone, tap Share and then
        &quot;Add to Home Screen&quot;.
      </>
    );
  }

  return (
    <div className="setting-section">
      <h1>App</h1>
      <div className="setting">
        <div>
          <h3>Install on this device</h3>
          <div className="sub-heading">{sub}</div>
        </div>
        <div>
          {canInstall() && !isInstalled() && (
            <Button onClick={() => promptInstall()}>Install</Button>
          )}
        </div>
      </div>
    </div>
  );
};

const roleNames: Record<GQL.UserRole, string> = {
  [GQL.UserRole.Admin]: "Admin",
  [GQL.UserRole.Editor]: "Editor",
  [GQL.UserRole.Viewer]: "Viewer",
};

export const AccountPanel: React.FC = () => {
  const Toast = useToast();
  const { me } = useCurrentUser();
  const [changePassword] = GQL.useChangeMyPasswordMutation();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [saving, setSaving] = useState(false);

  if (!me || !me.multi_user_enabled) {
    return (
      <>
        <div className="setting-section">
          <h1>Account</h1>
          <div className="sub-heading">
            No login is set up, so there is only one account.
          </div>
        </div>
        <InstallAppSection />
      </>
    );
  }

  async function onSave() {
    if (next !== confirm) {
      Toast.error("The new passwords do not match");
      return;
    }
    setSaving(true);
    try {
      await changePassword({
        variables: { current_password: current, new_password: next },
      });
      Toast.success("Password changed");
      setCurrent("");
      setNext("");
      setConfirm("");
    } catch (e) {
      Toast.error(e);
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
    <div className="setting-section stasharr-account">
      <h1>Account</h1>
      <div className="setting">
        <div>
          <h3>{me.username}</h3>
          <div className="sub-heading">
            {roleNames[me.role]}
            {me.tier ? ` · ${me.tier} tier` : ""}
            {me.is_owner ? " · owner" : ""}
          </div>
        </div>
        <div>
          <Button variant="secondary" href={`${baseURL}logout`}>
            Sign out
          </Button>
        </div>
      </div>

      {me.is_owner ? (
        <div className="setting">
          <div>
            <h3>Password</h3>
            <div className="sub-heading">
              The owner password is changed under Settings → Security.
            </div>
          </div>
        </div>
      ) : (
        <div className="setting stasharr-password">
          <div>
            <h3>Change password</h3>
          </div>
          <div>
            <Form.Control
              className="text-input mb-2"
              type="password"
              placeholder="Current password"
              autoComplete="current-password"
              value={current}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setCurrent(e.currentTarget.value)
              }
            />
            <Form.Control
              className="text-input mb-2"
              type="password"
              placeholder="New password"
              autoComplete="new-password"
              value={next}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setNext(e.currentTarget.value)
              }
            />
            <Form.Control
              className="text-input mb-2"
              type="password"
              placeholder="Repeat new password"
              autoComplete="new-password"
              value={confirm}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setConfirm(e.currentTarget.value)
              }
            />
            <Button
              disabled={saving || !current || next.length < 6}
              onClick={() => onSave()}
            >
              Change password
            </Button>
          </div>
        </div>
      )}
    </div>
    <InstallAppSection />
    </>
  );
};

export default AccountPanel;
