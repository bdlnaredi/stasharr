import React, { useState } from "react";
import { Badge, Button, Form, Table } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";

// Stasharr: invite links and tier upgrade requests, under Settings → Users.

function inviteLink(code: string) {
  const base = `${window.location.origin}${window.location.pathname.replace(/\/(settings|site).*$/, "/")}`;
  return `${base.replace(/\/$/, "")}/login?invite=${code}`;
}

function when(v?: string | null) {
  return v ? new Date(v).toLocaleDateString() : "never";
}

export const InvitesSection: React.FC<{ tiers: GQL.UserTierDataFragment[] }> = ({ tiers }) => {
  const Toast = useToast();
  const { data, refetch } = GQL.useUserInvitesQuery({ fetchPolicy: "network-only" });
  const [create] = GQL.useUserInviteCreateMutation();
  const [remove] = GQL.useUserInviteDeleteMutation();
  const [tier, setTier] = useState("");
  const [uses, setUses] = useState(1);
  const [days, setDays] = useState(7);
  const [role, setRole] = useState(GQL.UserRole.Viewer);
  const [note, setNote] = useState("");
  const invites = data?.userInvites ?? [];

  async function copy(code: string) {
    try {
      await navigator.clipboard.writeText(inviteLink(code));
      Toast.success("Invite link copied");
    } catch {
      window.prompt("Copy the invite link", inviteLink(code));
    }
  }

  return (
    <div className="setting-section">
      <h1>Invites</h1>
      <div className="sub-heading">
        Send someone a link to make their own account, in the tier you choose. The link works for the number of
        accounts and the days you set.
      </div>
      <div className="stasharr-invite-form">
        <Form.Control as="select" className="input-control" value={tier} onChange={(e) => setTier(e.currentTarget.value)}>
          <option value="">Whole library (no tier)</option>
          {tiers.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </Form.Control>
        <Form.Control
          as="select"
          className="input-control"
          value={role}
          onChange={(e) => setRole(e.currentTarget.value as GQL.UserRole)}
        >
          <option value={GQL.UserRole.Viewer}>Viewer</option>
          <option value={GQL.UserRole.Editor}>Editor</option>
        </Form.Control>
        <Form.Control as="select" className="input-control" value={uses} onChange={(e) => setUses(Number(e.currentTarget.value))}>
          {[1, 2, 5, 10, 25, 100].map((n) => (
            <option key={n} value={n}>
              {n === 1 ? "1 account" : `${n} accounts`}
            </option>
          ))}
        </Form.Control>
        <Form.Control as="select" className="input-control" value={days} onChange={(e) => setDays(Number(e.currentTarget.value))}>
          {[1, 3, 7, 30, 90, 0].map((n) => (
            <option key={n} value={n}>
              {n === 0 ? "No end" : n === 1 ? "1 day" : `${n} days`}
            </option>
          ))}
        </Form.Control>
        <Form.Control
          className="text-input"
          placeholder="Note, for example who it is for"
          value={note}
          onChange={(e) => setNote(e.currentTarget.value)}
        />
        <Button
          onClick={async () => {
            try {
              const res = await create({
                variables: { input: { tier_id: tier || undefined, role, max_uses: uses, days, note } },
              });
              setNote("");
              await refetch();
              const code = res.data?.userInviteCreate.code;
              if (code) await copy(code);
            } catch (e) {
              Toast.error(e);
            }
          }}
        >
          Make invite link
        </Button>
      </div>
      {invites.length > 0 && (
        <Table className="stasharr-table mt-3" responsive>
          <thead>
            <tr>
              <th>Invite</th>
              <th>Tier</th>
              <th>Used</th>
              <th>Ends</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {invites.map((i) => (
              <tr key={i.id} className={i.usable ? "" : "text-muted"}>
                <td>
                  {i.note || <code>{i.code.slice(0, 8)}…</code>}
                  {!i.usable && (
                    <Badge variant="secondary" className="ml-2">
                      done
                    </Badge>
                  )}
                </td>
                <td>
                  {i.tier?.name ?? "Whole library"}
                  {i.role === GQL.UserRole.Editor ? " · editor" : ""}
                </td>
                <td>
                  {i.uses} of {i.max_uses}
                </td>
                <td>{when(i.expires_at)}</td>
                <td className="text-right text-nowrap">
                  {i.usable && (
                    <Button size="sm" variant="secondary" onClick={() => copy(i.code)}>
                      Copy link
                    </Button>
                  )}{" "}
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={async () => {
                      try {
                        await remove({ variables: { id: i.id } });
                        await refetch();
                      } catch (e) {
                        Toast.error(e);
                      }
                    }}
                  >
                    Delete
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
      )}
    </div>
  );
};

export const TierRequestsSection: React.FC<{ onChange: () => void }> = ({ onChange }) => {
  const Toast = useToast();
  const { data, refetch } = GQL.useTierRequestsQuery({ fetchPolicy: "network-only" });
  const [decide] = GQL.useTierRequestDecideMutation();
  const list = data?.tierRequests ?? [];
  if (list.length === 0) return null;
  const pending = list.filter((r) => r.status === GQL.TierRequestStatus.Pending);

  async function answer(id: string, approve: boolean) {
    try {
      await decide({ variables: { id, approve } });
      await refetch();
      onChange();
      Toast.success(approve ? "Approved" : "Declined");
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div className="setting-section">
      <h1>
        Upgrade requests{" "}
        {pending.length > 0 && <Badge variant="primary">{pending.length}</Badge>}
      </h1>
      <Table className="stasharr-table" responsive>
        <tbody>
          {list.slice(0, 20).map((r) => (
            <tr key={r.id} className={r.status === GQL.TierRequestStatus.Pending ? "" : "text-muted"}>
              <td>
                <strong>{r.username}</strong> asks for <strong>{r.tier.name}</strong>
                {r.message && <div className="small">“{r.message}”</div>}
                <div className="small text-muted">{new Date(r.created_at).toLocaleString()}</div>
              </td>
              <td className="text-right text-nowrap">
                {r.status === GQL.TierRequestStatus.Pending ? (
                  <>
                    <Button size="sm" onClick={() => answer(r.id, true)}>
                      Approve
                    </Button>{" "}
                    <Button size="sm" variant="secondary" onClick={() => answer(r.id, false)}>
                      Decline
                    </Button>
                  </>
                ) : r.status === GQL.TierRequestStatus.Approved ? (
                  "Approved"
                ) : (
                  "Declined"
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </Table>
    </div>
  );
};
