import React, { useEffect, useState } from "react";
import { Button, Card, Form } from "react-bootstrap";
import { faPaperPlane, faPlus, faTrashAlt } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";

type Target = Omit<GQL.NotifyTargetDataFragment, "__typename" | "id"> & {
  id?: string;
};

const kinds: { id: string; label: string; help: string }[] = [
  {
    id: "discord",
    label: "Discord",
    help: "In Discord: channel settings → Integrations → Webhooks → New webhook → Copy webhook URL.",
  },
  {
    id: "telegram",
    label: "Telegram",
    help: "Create a bot with @BotFather to get the token, send your bot a message, then get your chat id from @userinfobot.",
  },
  {
    id: "ntfy",
    label: "ntfy",
    help: "The topic address, for example https://ntfy.sh/my-stasharr. Install the ntfy app and subscribe to the same topic.",
  },
  {
    id: "gotify",
    label: "Gotify",
    help: "Your Gotify server address and an app token from Gotify → Apps.",
  },
  {
    id: "webhook",
    label: "Webhook",
    help: 'Any address. Stasharr sends a JSON POST with "event", "title", "message" and "time" (Home Assistant, n8n and so on).',
  },
];

function emptyTarget(events: string[]): Target {
  return {
    name: "",
    kind: "discord",
    enabled: true,
    url: "",
    token: "",
    chat_id: "",
    events,
  };
}

function toInput(t: Target): GQL.NotifyTargetInput {
  return {
    id: t.id,
    name: t.name,
    kind: t.kind,
    enabled: t.enabled,
    url: t.url,
    token: t.token,
    chat_id: t.chat_id,
    events: t.events,
  };
}

export const NotificationsPanel: React.FC = () => {
  const Toast = useToast();
  const { data, loading, refetch } = GQL.useNotifySettingsQuery({
    fetchPolicy: "network-only",
  });
  const [save] = GQL.useSaveNotifyTargetsMutation();
  const [test] = GQL.useTestNotifyTargetMutation();
  const [targets, setTargets] = useState<Target[]>([]);
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (data) {
      setTargets(
        data.notifyTargets.map((t) => ({
          id: t.id,
          name: t.name,
          kind: t.kind,
          enabled: t.enabled,
          url: t.url,
          token: t.token,
          chat_id: t.chat_id,
          events: t.events,
        }))
      );
      setDirty(false);
    }
  }, [data]);

  if (loading && !data) return <LoadingIndicator />;
  if (!data) return null;

  const events = data.notifyEvents;

  function change(i: number, patch: Partial<Target>) {
    setTargets((list) => list.map((t, j) => (j === i ? { ...t, ...patch } : t)));
    setDirty(true);
  }

  function toggleEvent(i: number, id: string) {
    const t = targets[i];
    const has = t.events.includes(id);
    change(i, { events: has ? t.events.filter((e) => e !== id) : [...t.events, id] });
  }

  async function onSave() {
    setBusy(true);
    try {
      await save({ variables: { targets: targets.map(toInput) } });
      Toast.success("Notifications saved");
      await refetch();
    } catch (e) {
      Toast.error(e);
    } finally {
      setBusy(false);
    }
  }

  async function onTest(t: Target) {
    setBusy(true);
    try {
      await test({ variables: { target: toInput(t) } });
      Toast.success(`Test message sent to ${t.name || t.kind}`);
    } catch (e) {
      Toast.error(e);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="setting-section stasharr-notify">
      <div className="stasharr-section-head">
        <h1>Notifications</h1>
        <Button
          variant="secondary"
          onClick={() => {
            setTargets((list) => [
              ...list,
              emptyTarget(events.map((e) => e.id).filter((e) => e !== "job_failed")),
            ]);
            setDirty(true);
          }}
        >
          <Icon icon={faPlus} /> Add
        </Button>
      </div>
      <div className="sub-heading">
        Get a message on your phone or in a chat when new scenes arrive, when
        a scene cannot be identified, when a task fails, or when a backup or
        metadata refresh finishes.
      </div>

      {targets.length === 0 && (
        <div className="sub-heading mt-3">No notifications set up yet.</div>
      )}

      {targets.map((t, i) => {
        const kind = kinds.find((k) => k.id === t.kind) ?? kinds[0];
        return (
          <Card key={t.id ?? `new-${i}`} className="stasharr-notify-card">
            <Card.Body>
              <div className="stasharr-notify-row">
                <Form.Control
                  className="text-input"
                  placeholder="Name"
                  value={t.name}
                  onChange={(e) => change(i, { name: e.currentTarget.value })}
                />
                <Form.Control
                  as="select"
                  className="input-control"
                  value={t.kind}
                  onChange={(e) => change(i, { kind: e.currentTarget.value })}
                >
                  {kinds.map((k) => (
                    <option key={k.id} value={k.id}>
                      {k.label}
                    </option>
                  ))}
                </Form.Control>
                <Form.Switch
                  id={`notify-enabled-${i}`}
                  label="On"
                  checked={t.enabled}
                  onChange={() => change(i, { enabled: !t.enabled })}
                />
              </div>
              <div className="sub-heading">{kind.help}</div>
              {t.kind !== "telegram" && (
                <Form.Control
                  className="text-input mt-2"
                  placeholder={
                    t.kind === "discord"
                      ? "https://discord.com/api/webhooks/…"
                      : t.kind === "ntfy"
                      ? "https://ntfy.sh/my-topic"
                      : "https://…"
                  }
                  value={t.url}
                  onChange={(e) => change(i, { url: e.currentTarget.value })}
                />
              )}
              {t.kind !== "discord" && (
                <Form.Control
                  className="text-input mt-2"
                  type="password"
                  autoComplete="off"
                  placeholder={
                    t.kind === "telegram"
                      ? "Bot token"
                      : t.kind === "gotify"
                      ? "App token"
                      : "Access token (optional)"
                  }
                  value={t.token}
                  onChange={(e) => change(i, { token: e.currentTarget.value })}
                />
              )}
              {t.kind === "telegram" && (
                <Form.Control
                  className="text-input mt-2"
                  placeholder="Chat id"
                  value={t.chat_id}
                  onChange={(e) => change(i, { chat_id: e.currentTarget.value })}
                />
              )}
              <div className="stasharr-notify-events">
                {events.map((e) => (
                  <Form.Check
                    key={e.id}
                    id={`notify-${i}-${e.id}`}
                    label={e.label}
                    checked={t.events.includes(e.id)}
                    onChange={() => toggleEvent(i, e.id)}
                  />
                ))}
              </div>
              <div className="text-right">
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={busy}
                  onClick={() => onTest(t)}
                >
                  <Icon icon={faPaperPlane} /> Send test
                </Button>{" "}
                <Button
                  variant="danger"
                  size="sm"
                  disabled={busy}
                  title="Remove"
                  onClick={() => {
                    setTargets((list) => list.filter((_, j) => j !== i));
                    setDirty(true);
                  }}
                >
                  <Icon icon={faTrashAlt} />
                </Button>
              </div>
            </Card.Body>
          </Card>
        );
      })}

      {dirty && (
        <div className="mt-3">
          <Button variant="primary" disabled={busy} onClick={() => onSave()}>
            Save notifications
          </Button>{" "}
          <Button variant="secondary" disabled={busy} onClick={() => refetch()}>
            Discard changes
          </Button>
        </div>
      )}
    </div>
  );
};
