import React, { useState } from "react";
import { Button, Form } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";

// Stasharr: two-step sign in with an authenticator app, and asking the
// admin for another tier. Both sit on the Account page (site and admin).

const RecoveryCodes: React.FC<{ codes: string[]; onDone: () => void }> = ({ codes, onDone }) => (
  <div className="stasharr-recovery">
    <div className="sub-heading">
      Keep these recovery codes somewhere safe. Each one signs you in once if you lose your phone. They are not
      shown again.
    </div>
    <pre>{codes.join("\n")}</pre>
    <Button
      variant="secondary"
      size="sm"
      onClick={() => {
        navigator.clipboard?.writeText(codes.join("\n")).catch(() => {});
      }}
    >
      Copy
    </Button>{" "}
    <Button size="sm" onClick={onDone}>
      I saved them
    </Button>
  </div>
);

export const TwoStepSection: React.FC = () => {
  const Toast = useToast();
  const status = GQL.useMyTwoStepQuery({ fetchPolicy: "network-only" });
  const [start] = GQL.useTwoStepStartMutation();
  const [confirm] = GQL.useTwoStepConfirmMutation();
  const [disable] = GQL.useTwoStepDisableMutation();
  const [newCodes] = GQL.useTwoStepRecoveryCodesMutation();
  const [forget] = GQL.useTwoStepForgetDevicesMutation();
  const [setup, setSetup] = useState<GQL.TwoStepStartMutation["twoStepStart"] | null>(null);
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [codes, setCodes] = useState<string[] | null>(null);
  const [busy, setBusy] = useState(false);
  const st = status.data?.myTwoStep;

  async function run<T>(fn: () => Promise<T>): Promise<T | undefined> {
    setBusy(true);
    try {
      return await fn();
    } catch (e) {
      Toast.error(e);
      return undefined;
    } finally {
      setBusy(false);
    }
  }

  if (!st) return null;

  return (
    <div className="setting-section">
      <h1>Two-step sign in</h1>
      <div className="sub-heading">
        After your password, Stasharr also asks for a 6-digit code from an authenticator app on your phone (Google
        Authenticator, Microsoft Authenticator, Aegis…). A device you sign in on with a code is remembered for 90
        days.
      </div>

      {codes && <RecoveryCodes codes={codes} onDone={() => setCodes(null)} />}

      {!st.enabled && !setup && (
        <Button
          className="mt-2"
          disabled={busy}
          onClick={() =>
            run(async () => {
              const res = await start();
              setSetup(res.data?.twoStepStart ?? null);
              setCode("");
            })
          }
        >
          Turn on two-step sign in
        </Button>
      )}

      {!st.enabled && setup && (
        <div className="stasharr-two-step-setup">
          <div
            className="stasharr-two-step-qr"
            // the server made this picture of the setup link
            dangerouslySetInnerHTML={{ __html: setup.qr_svg }}
          />
          <div>
            <ol>
              <li>Open your authenticator app and add an account by scanning this code.</li>
              <li>
                On this phone instead? <a href={setup.url}>Open in authenticator app</a>, or type this key:{" "}
                <code>{setup.secret.replace(/(.{4})/g, "$1 ").trim()}</code>
              </li>
              <li>Type the 6-digit code the app shows:</li>
            </ol>
            <Form.Control
              className="text-input"
              inputMode="numeric"
              autoComplete="one-time-code"
              placeholder="123456"
              value={code}
              onChange={(e) => setCode(e.currentTarget.value)}
            />
            <div className="mt-2">
              <Button
                disabled={busy || code.replace(/\s/g, "").length !== 6}
                onClick={() =>
                  run(async () => {
                    const res = await confirm({ variables: { code } });
                    setCodes(res.data?.twoStepConfirm ?? null);
                    setSetup(null);
                    await status.refetch();
                    Toast.success("Two-step sign in is on");
                  })
                }
              >
                Turn on
              </Button>{" "}
              <Button variant="secondary" onClick={() => setSetup(null)}>
                Cancel
              </Button>
            </div>
          </div>
        </div>
      )}

      {st.enabled && (
        <>
          <div className="setting">
            <div>
              <h3>On</h3>
              <div className="sub-heading">
                {st.recovery_left} recovery code(s) left · {st.trusted_devices} remembered device(s)
              </div>
            </div>
            <div>
              <Button
                variant="secondary"
                size="sm"
                disabled={busy}
                onClick={() =>
                  run(async () => {
                    const res = await newCodes();
                    setCodes(res.data?.twoStepRecoveryCodes ?? null);
                    await status.refetch();
                  })
                }
              >
                New recovery codes
              </Button>{" "}
              <Button
                variant="secondary"
                size="sm"
                disabled={busy || st.trusted_devices === 0}
                onClick={() =>
                  run(async () => {
                    await forget();
                    await status.refetch();
                    Toast.success("Every device asks for a code next time");
                  })
                }
              >
                Forget devices
              </Button>
            </div>
          </div>
          <div className="setting">
            <div>
              <h3>Turn off</h3>
              <div className="sub-heading">Needs your password.</div>
            </div>
            <div className="d-flex">
              <Form.Control
                className="text-input mr-2"
                type="password"
                placeholder="Password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.currentTarget.value)}
              />
              <Button
                variant="danger"
                disabled={busy || !password}
                onClick={() =>
                  run(async () => {
                    await disable({ variables: { password } });
                    setPassword("");
                    await status.refetch();
                    Toast.success("Two-step sign in is off");
                  })
                }
              >
                Turn off
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export const TierRequestSection: React.FC = () => {
  const Toast = useToast();
  const { data, refetch } = GQL.useTierChoicesQuery({ fetchPolicy: "network-only" });
  const [create] = GQL.useTierRequestCreateMutation();
  const [cancel] = GQL.useTierRequestCancelMutation();
  const [tier, setTier] = useState("");
  const [message, setMessage] = useState("");
  const choices = data?.tierChoices ?? [];
  const others = choices.filter((c) => !c.current);
  if (others.length === 0) return null;
  const pending = data?.myTierRequests.find((r) => r.status === GQL.TierRequestStatus.Pending);
  const last = data?.myTierRequests.find((r) => r.status !== GQL.TierRequestStatus.Pending);
  const current = choices.find((c) => c.current);

  return (
    <div className="setting-section">
      <h1>Tier</h1>
      <div className="sub-heading">
        Your tier decides what you can see{current ? `. You are in ${current.name}` : ""}. Ask the admin for
        another one.
      </div>
      {pending ? (
        <div className="setting">
          <div>
            <h3>Asked for {pending.tier.name}</h3>
            <div className="sub-heading">Waiting for the admin{pending.message ? `: “${pending.message}”` : ""}</div>
          </div>
          <div>
            <Button
              variant="secondary"
              size="sm"
              onClick={async () => {
                try {
                  await cancel();
                  await refetch();
                } catch (e) {
                  Toast.error(e);
                }
              }}
            >
              Cancel request
            </Button>
          </div>
        </div>
      ) : (
        <div className="stasharr-tier-request">
          {last && (
            <div className="sub-heading mb-2">
              Your request for {last.tier.name} was {last.status === GQL.TierRequestStatus.Approved ? "approved" : "declined"}.
            </div>
          )}
          <Form.Control as="select" className="input-control mb-2" value={tier} onChange={(e) => setTier(e.currentTarget.value)}>
            <option value="">Choose a tier…</option>
            {others.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
                {c.description ? ` – ${c.description}` : ""}
              </option>
            ))}
          </Form.Control>
          <Form.Control
            as="textarea"
            rows={2}
            className="text-input mb-2"
            placeholder="A note for the admin (optional)"
            value={message}
            onChange={(e) => setMessage(e.currentTarget.value)}
          />
          <Button
            disabled={!tier}
            onClick={async () => {
              try {
                await create({ variables: { tier_id: tier, message } });
                setTier("");
                setMessage("");
                await refetch();
                Toast.success("Request sent");
              } catch (e) {
                Toast.error(e);
              }
            }}
          >
            Ask for this tier
          </Button>
        </div>
      )}
    </div>
  );
};
