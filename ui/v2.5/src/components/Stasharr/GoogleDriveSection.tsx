import React, { useEffect, useState } from "react";
import { Button, Form, Table } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { SettingRow, formatBytes, formatDate } from "./util";

// Backups to Google Drive: the admin's own Google OAuth client, a sign in
// with a code on google.com/device, then every scheduled backup is uploaded.

const Setup: React.FC = () => (
  <details className="stasharr-help mt-2">
    <summary>How to get a client ID (once, about 5 minutes)</summary>
    <ol>
      <li>
        Open{" "}
        <a href="https://console.cloud.google.com/" target="_blank" rel="noreferrer">
          console.cloud.google.com
        </a>{" "}
        with the Google account whose Drive should hold the backups, and make a new project (for
        example "Stasharr").
      </li>
      <li>
        <b>APIs &amp; Services → Library</b>: search for <b>Google Drive API</b> and press{" "}
        <b>Enable</b>.
      </li>
      <li>
        <b>APIs &amp; Services → OAuth consent screen</b> (Google Auth Platform): choose{" "}
        <b>External</b>, give it a name and your email. Under <b>Audience</b>, press{" "}
        <b>Publish app</b> so the status is <b>In production</b>. In "Testing" Google signs the
        server out every 7 days.
      </li>
      <li>
        <b>Credentials → Create credentials → OAuth client ID</b>, application type{" "}
        <b>TVs and Limited Input devices</b>. Copy the client ID and client secret here.
      </li>
      <li>
        Press <b>Connect Google Drive</b>, open the link and enter the code. Google says the app
        is not verified: choose <b>Advanced → Go to (your app)</b>. It is your own app, and it can
        only see the files Stasharr puts on your Drive.
      </li>
    </ol>
  </details>
);

export const GoogleDriveSection: React.FC<{
  latestBackupID?: string;
}> = ({ latestBackupID }) => {
  const Toast = useToast();
  const { data, refetch, startPolling, stopPolling } = GQL.useGoogleDriveBackupQuery({
    fetchPolicy: "network-only",
  });
  const files = GQL.useGoogleDriveFilesQuery({
    fetchPolicy: "network-only",
    skip: !data?.googleDriveBackup.connected,
  });
  const [configure] = GQL.useConfigureGoogleDriveMutation();
  const [startSignIn] = GQL.useStartGoogleDriveSignInMutation();
  const [cancelSignIn] = GQL.useCancelGoogleDriveSignInMutation();
  const [disconnect] = GQL.useDisconnectGoogleDriveMutation();
  const [upload] = GQL.useUploadBackupToGoogleDriveMutation();
  const [download] = GQL.useDownloadGoogleDriveBackupMutation();
  const [busy, setBusy] = useState(false);
  const [clientID, setClientID] = useState("");
  const [secret, setSecret] = useState("");

  const g = data?.googleDriveBackup;
  const waiting = !!g?.sign_in;

  useEffect(() => {
    if (g) setClientID(g.client_id);
  }, [g?.client_id]); // eslint-disable-line react-hooks/exhaustive-deps

  // while a code is shown, check every few seconds whether it was entered
  useEffect(() => {
    if (waiting) startPolling(3000);
    else stopPolling();
    return () => stopPolling();
  }, [waiting, startPolling, stopPolling]);

  useEffect(() => {
    if (g?.connected) files.refetch().catch(() => {});
  }, [g?.connected]); // eslint-disable-line react-hooks/exhaustive-deps

  if (!g) return null;

  async function run(fn: () => Promise<unknown>, success?: string) {
    setBusy(true);
    try {
      await fn();
      if (success) Toast.success(success);
      await refetch();
    } catch (e) {
      Toast.error(e);
    } finally {
      setBusy(false);
    }
  }

  function saveClient() {
    return configure({
      variables: { input: { client_id: clientID.trim(), client_secret: secret.trim() || undefined } },
    }).then(() => setSecret(""));
  }

  const clientChanged = clientID.trim() !== g.client_id || secret.trim() !== "";
  const canConnect = clientID.trim() !== "" && (g.has_client_secret || secret.trim() !== "");

  return (
    <div className="setting-section">
      <div className="stasharr-section-head">
        <h1>Google Drive</h1>
        {g.connected && latestBackupID && (
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() =>
              run(
                () => upload({ variables: { id: latestBackupID } }),
                "Uploading the newest backup. See Settings → Tasks for progress."
              )
            }
          >
            Upload newest backup
          </Button>
        )}
      </div>

      {!g.connected && (
        <>
          <div className="sub-heading mb-2">
            Keep a copy of every scheduled backup on your Google Drive, in a folder called "
            {g.folder_name}", so a broken disk does not take your library with it.
          </div>
          <SettingRow heading="Client ID" sub="From your Google Cloud project (see below)">
            <Form.Control
              className="text-input"
              value={clientID}
              placeholder="1234…apps.googleusercontent.com"
              disabled={busy || waiting}
              onChange={(e) => setClientID(e.currentTarget.value)}
            />
          </SettingRow>
          <SettingRow heading="Client secret" sub={g.has_client_secret ? "Saved. Type a new one to replace it." : undefined}>
            <Form.Control
              className="text-input"
              type="password"
              value={secret}
              placeholder={g.has_client_secret ? "••••••••" : "GOCSPX-…"}
              disabled={busy || waiting}
              onChange={(e) => setSecret(e.currentTarget.value)}
            />
          </SettingRow>

          {waiting && g.sign_in ? (
            <div className="stasharr-gdrive-code">
              <div>
                Open{" "}
                <a href={g.sign_in.verification_url} target="_blank" rel="noreferrer">
                  {g.sign_in.verification_url}
                </a>{" "}
                and enter this code:
              </div>
              <div className="stasharr-gdrive-usercode">{g.sign_in.user_code}</div>
              <div className="sub-heading">Waiting for you to approve… this page updates by itself.</div>
              <Button variant="secondary" size="sm" disabled={busy} onClick={() => run(() => cancelSignIn())}>
                Cancel
              </Button>
            </div>
          ) : (
            <Button
              className="mt-2"
              variant="primary"
              disabled={busy || !canConnect}
              onClick={() =>
                run(async () => {
                  if (clientChanged) await saveClient();
                  await startSignIn();
                })
              }
            >
              Connect Google Drive
            </Button>
          )}
          {g.sign_in_error && !waiting && <div className="text-danger mt-2">{g.sign_in_error}</div>}
          <Setup />
        </>
      )}

      {g.connected && (
        <>
          <SettingRow
            heading="Connected"
            sub={`${g.email ?? "Google account"} · folder "${g.folder_name}"`}
          >
            <Button
              variant="danger"
              size="sm"
              disabled={busy}
              onClick={() => {
                if (!window.confirm("Disconnect Google Drive? Backups already on Drive stay there.")) return;
                run(() => disconnect(), "Google Drive disconnected");
              }}
            >
              Disconnect
            </Button>
          </SettingRow>
          <SettingRow
            heading="Upload scheduled backups"
            sub={
              g.last_upload
                ? `Last upload ${formatDate(g.last_upload)}.`
                : "Each daily backup is uploaded after it is made."
            }
          >
            <Form.Switch
              id="gdrive-enabled"
              disabled={busy}
              checked={g.enabled}
              onChange={() => run(() => configure({ variables: { input: { enabled: !g.enabled } } }))}
            />
          </SettingRow>
          <SettingRow heading="Backups to keep on Google Drive" sub="Older ones on Drive are deleted after each upload.">
            <Form.Control
              as="select"
              className="input-control"
              disabled={busy}
              value={g.keep}
              onChange={(e) =>
                run(() => configure({ variables: { input: { keep: Number(e.currentTarget.value) } } }))
              }
            >
              {[3, 5, 7, 14, 30, 60, 90].map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </Form.Control>
          </SettingRow>
          {g.last_error && <div className="text-danger mt-2">Last upload failed: {g.last_error}</div>}

          {files.error && <div className="text-danger mt-2">{files.error.message}</div>}
          {(files.data?.googleDriveFiles.length ?? 0) > 0 && (
            <Table responsive size="sm" className="stasharr-table mt-3">
              <thead>
                <tr>
                  <th>On Google Drive</th>
                  <th>Size</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {files.data?.googleDriveFiles.map((f) => (
                  <tr key={f.id}>
                    <td>
                      <div>{formatDate(f.created_at)}</div>
                      <small className="text-muted">{f.name}</small>
                    </td>
                    <td>{formatBytes(f.size)}</td>
                    <td className="text-right text-nowrap">
                      <Button
                        variant="secondary"
                        size="sm"
                        disabled={busy}
                        title="Copy it to the backup folder; it then shows in the list above and can be restored"
                        onClick={() =>
                          run(
                            () => download({ variables: { id: f.id } }),
                            "Copying from Google Drive. It appears in the list of backups when done."
                          )
                        }
                      >
                        Copy to server
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          )}
        </>
      )}
    </div>
  );
};
