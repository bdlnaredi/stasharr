import React, { useState } from "react";
import { Badge, Button, Form, Table } from "react-bootstrap";
import {
  faKey,
  faPencilAlt,
  faPlus,
  faTrashAlt,
} from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { ModalComponent } from "src/components/Shared/Modal";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { TagIDSelect } from "src/components/Tags/TagSelect";
import { useToast } from "src/hooks/Toast";
import { useCurrentUser } from "./CurrentUser";

const roleLabels: Record<GQL.UserRole, string> = {
  [GQL.UserRole.Admin]: "Admin",
  [GQL.UserRole.Editor]: "Editor",
  [GQL.UserRole.Viewer]: "Viewer",
};

const roleDescriptions: Record<GQL.UserRole, string> = {
  [GQL.UserRole.Admin]: "Everything, including settings and users",
  [GQL.UserRole.Editor]:
    "Edit metadata, scrape, identify and run library tasks",
  [GQL.UserRole.Viewer]:
    "Watch what their tier allows, with their own ratings and history",
};

const allRoles = [GQL.UserRole.Viewer, GQL.UserRole.Editor, GQL.UserRole.Admin];

function formatDate(value?: string | null) {
  if (!value) return "Never";
  const d = new Date(value);
  return d.toLocaleString();
}

interface IUserForm {
  id?: string;
  username: string;
  password: string;
  role: GQL.UserRole;
  tier_id: string;
  disabled: boolean;
}

const emptyUser: IUserForm = {
  username: "",
  password: "",
  role: GQL.UserRole.Viewer,
  tier_id: "",
  disabled: false,
};

const UserDialog: React.FC<{
  user: IUserForm;
  tiers: GQL.UserTierDataFragment[];
  onClose: (saved: boolean) => void;
}> = ({ user, tiers, onClose }) => {
  const Toast = useToast();
  const [value, setValue] = useState<IUserForm>(user);
  const [saving, setSaving] = useState(false);
  const [createUser] = GQL.useUserCreateMutation();
  const [updateUser] = GQL.useUserUpdateMutation();
  const isNew = !user.id;

  function set(v: Partial<IUserForm>) {
    setValue({ ...value, ...v });
  }

  async function save() {
    setSaving(true);
    try {
      if (isNew) {
        await createUser({
          variables: {
            input: {
              username: value.username,
              password: value.password,
              role: value.role,
              tier_id: value.tier_id || null,
              disabled: value.disabled,
            },
          },
        });
        Toast.success(`Created ${value.username}`);
      } else {
        await updateUser({
          variables: {
            input: {
              id: user.id!,
              username: value.username,
              password: value.password || undefined,
              role: value.role,
              tier_id: value.tier_id || null,
              disabled: value.disabled,
            },
          },
        });
        Toast.success(`Saved ${value.username}`);
      }
      onClose(true);
    } catch (e) {
      Toast.error(e);
    } finally {
      setSaving(false);
    }
  }

  const valid =
    value.username.trim() !== "" && (!isNew || value.password.length >= 6);

  return (
    <ModalComponent
      show
      header={isNew ? "Add user" : `Edit ${user.username}`}
      isRunning={saving}
      disabled={!valid}
      accept={{ text: "Save", onClick: save }}
      cancel={{ text: "Cancel", variant: "secondary", onClick: () => onClose(false) }}
      onHide={() => onClose(false)}
    >
      <Form.Group>
        <Form.Label>Username</Form.Label>
        <Form.Control
          className="text-input"
          value={value.username}
          autoComplete="off"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            set({ username: e.currentTarget.value })
          }
        />
      </Form.Group>
      <Form.Group>
        <Form.Label>{isNew ? "Password" : "New password"}</Form.Label>
        <Form.Control
          className="text-input"
          type="password"
          autoComplete="new-password"
          placeholder={isNew ? "At least 6 characters" : "Leave empty to keep"}
          value={value.password}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            set({ password: e.currentTarget.value })
          }
        />
      </Form.Group>
      <Form.Group>
        <Form.Label>Role</Form.Label>
        {allRoles.map((r) => (
          <Form.Check
            key={r}
            type="radio"
            id={`role-${r}`}
            name="role"
            checked={value.role === r}
            onChange={() => set({ role: r })}
            label={
              <span>
                <strong>{roleLabels[r]}</strong>{" "}
                <small className="text-muted">{roleDescriptions[r]}</small>
              </span>
            }
          />
        ))}
      </Form.Group>
      <Form.Group>
        <Form.Label>Tier</Form.Label>
        <Form.Control
          as="select"
          className="input-control"
          value={value.tier_id}
          onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
            set({ tier_id: e.currentTarget.value })
          }
        >
          <option value="">No tier (whole library)</option>
          {tiers.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </Form.Control>
        <Form.Text className="text-muted">
          Admins always see the whole library.
        </Form.Text>
      </Form.Group>
      <Form.Check
        id="user-disabled"
        type="switch"
        checked={value.disabled}
        onChange={() => set({ disabled: !value.disabled })}
        label="Disabled (cannot sign in)"
      />
    </ModalComponent>
  );
};

interface ITierForm {
  id?: string;
  name: string;
  description: string;
  allowed_paths: string;
  blocked_tag_ids: string[];
  required_tag_ids: string[];
  can_download: boolean;
  can_view_reels: boolean;
}

const emptyTier: ITierForm = {
  name: "",
  description: "",
  allowed_paths: "",
  blocked_tag_ids: [],
  required_tag_ids: [],
  can_download: true,
  can_view_reels: true,
};

const TierDialog: React.FC<{
  tier: ITierForm;
  libraryPaths: string[];
  onClose: (saved: boolean) => void;
}> = ({ tier, libraryPaths, onClose }) => {
  const Toast = useToast();
  const [value, setValue] = useState<ITierForm>(tier);
  const [saving, setSaving] = useState(false);
  const [createTier] = GQL.useUserTierCreateMutation();
  const [updateTier] = GQL.useUserTierUpdateMutation();
  const isNew = !tier.id;

  function set(v: Partial<ITierForm>) {
    setValue({ ...value, ...v });
  }

  const paths = value.allowed_paths
    .split("\n")
    .map((p) => p.trim())
    .filter((p) => p !== "");

  function togglePath(p: string) {
    const next = paths.includes(p)
      ? paths.filter((x) => x !== p)
      : [...paths, p];
    set({ allowed_paths: next.join("\n") });
  }

  async function save() {
    setSaving(true);
    const input = {
      name: value.name,
      description: value.description,
      allowed_paths: paths,
      blocked_tag_ids: value.blocked_tag_ids,
      required_tag_ids: value.required_tag_ids,
      can_download: value.can_download,
      can_view_reels: value.can_view_reels,
    };
    try {
      if (isNew) {
        await createTier({ variables: { input } });
      } else {
        await updateTier({ variables: { input: { id: tier.id!, ...input } } });
      }
      Toast.success(`Saved tier ${value.name}`);
      onClose(true);
    } catch (e) {
      Toast.error(e);
    } finally {
      setSaving(false);
    }
  }

  return (
    <ModalComponent
      show
      header={isNew ? "Add tier" : `Edit tier ${tier.name}`}
      isRunning={saving}
      disabled={value.name.trim() === ""}
      accept={{ text: "Save", onClick: save }}
      cancel={{ text: "Cancel", variant: "secondary", onClick: () => onClose(false) }}
      onHide={() => onClose(false)}
    >
      <Form.Group>
        <Form.Label>Name</Form.Label>
        <Form.Control
          className="text-input"
          placeholder="For example Free, Premium or VIP"
          value={value.name}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            set({ name: e.currentTarget.value })
          }
        />
      </Form.Group>
      <Form.Group>
        <Form.Label>Description</Form.Label>
        <Form.Control
          className="text-input"
          value={value.description}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            set({ description: e.currentTarget.value })
          }
        />
      </Form.Group>
      <Form.Group>
        <Form.Label>Library access</Form.Label>
        <Form.Text className="text-muted mb-2">
          Tick the library folders this tier can see. Leave all unticked for
          the whole library.
        </Form.Text>
        {libraryPaths.map((p) => (
          <Form.Check
            key={p}
            id={`tier-path-${p}`}
            checked={paths.includes(p)}
            onChange={() => togglePath(p)}
            label={<code>{p}</code>}
          />
        ))}
        <Form.Label className="mt-2">
          <small>Sub-folders (one per line)</small>
        </Form.Label>
        <Form.Control
          as="textarea"
          rows={3}
          className="text-input"
          placeholder="/data/videos/premium"
          value={value.allowed_paths}
          onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
            set({ allowed_paths: e.currentTarget.value })
          }
        />
      </Form.Group>
      <Form.Group>
        <Form.Label>Hide content with these tags</Form.Label>
        <TagIDSelect
          isMulti
          ids={value.blocked_tag_ids}
          onSelect={(items) => set({ blocked_tag_ids: items.map((t) => t.id) })}
        />
      </Form.Group>
      <Form.Group>
        <Form.Label>Only show content with one of these tags</Form.Label>
        <TagIDSelect
          isMulti
          ids={value.required_tag_ids}
          onSelect={(items) =>
            set({ required_tag_ids: items.map((t) => t.id) })
          }
        />
        <Form.Text className="text-muted">Leave empty for no limit.</Form.Text>
      </Form.Group>
      <Form.Check
        id="tier-download"
        type="switch"
        checked={value.can_download}
        onChange={() => set({ can_download: !value.can_download })}
        label="Can download files"
      />
      <Form.Check
        id="tier-reels"
        type="switch"
        checked={value.can_view_reels}
        onChange={() => set({ can_view_reels: !value.can_view_reels })}
        label="Can watch reels"
      />
    </ModalComponent>
  );
};

const ApiKeyDialog: React.FC<{
  user: GQL.UserDataFragment;
  onClose: () => void;
}> = ({ user, onClose }) => {
  const Toast = useToast();
  const [generate] = GQL.useUserGenerateApiKeyMutation();
  const [key, setKey] = useState<string>();

  async function run(clear: boolean) {
    try {
      const res = await generate({ variables: { id: user.id, clear } });
      if (clear) {
        Toast.success("API key removed");
        onClose();
      } else {
        setKey(res.data?.userGenerateAPIKey ?? undefined);
      }
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <ModalComponent
      show
      header={`API key for ${user.username}`}
      accept={{ text: "Close", onClick: onClose }}
      onHide={onClose}
    >
      {key ? (
        <>
          <p>Copy this key now. It will not be shown again.</p>
          <Form.Control className="text-input" readOnly value={key} />
        </>
      ) : (
        <>
          <p>
            An API key lets apps and scripts act as this user. Making a new key
            replaces the old one.
          </p>
          <Button className="mr-2" onClick={() => run(false)}>
            Make new key
          </Button>
          {user.has_api_key && (
            <Button variant="danger" onClick={() => run(true)}>
              Remove key
            </Button>
          )}
        </>
      )}
    </ModalComponent>
  );
};

export const UsersPanel: React.FC = () => {
  const Toast = useToast();
  const { me } = useCurrentUser();
  const users = GQL.useUsersQuery({ fetchPolicy: "network-only" });
  const tiers = GQL.useUserTiersQuery({ fetchPolicy: "network-only" });
  const config = GQL.useConfigurationQuery();
  const [destroyUser] = GQL.useUserDestroyMutation();
  const [destroyTier] = GQL.useUserTierDestroyMutation();

  const [editUser, setEditUser] = useState<IUserForm>();
  const [editTier, setEditTier] = useState<ITierForm>();
  const [apiKeyUser, setApiKeyUser] = useState<GQL.UserDataFragment>();

  const libraryPaths = (config.data?.configuration.general.stashes ?? []).map(
    (s) => s.path
  );

  if (me && !me.multi_user_enabled) {
    return (
      <div className="setting-section">
        <h1>Users</h1>
        <div className="setting">
          <div>
            <h3>Turn on logins first</h3>
            <div className="sub-heading">
              Set an owner username and password under Settings → Security.
              That account is the owner and always has full access. Then come
              back here to add more people.
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (users.loading || tiers.loading) return <LoadingIndicator />;

  const userList = users.data?.users ?? [];
  const tierList = tiers.data?.userTiers ?? [];

  async function onDeleteUser(u: GQL.UserDataFragment) {
    if (!window.confirm(`Delete ${u.username}? Their history and ratings will be removed.`)) {
      return;
    }
    try {
      await destroyUser({ variables: { id: u.id } });
      users.refetch();
      tiers.refetch();
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onDeleteTier(t: GQL.UserTierDataFragment) {
    if (!window.confirm(`Delete tier ${t.name}? Its users will see the whole library.`)) {
      return;
    }
    try {
      await destroyTier({ variables: { id: t.id } });
      users.refetch();
      tiers.refetch();
    } catch (e) {
      Toast.error(e);
    }
  }

  function refresh(saved: boolean) {
    setEditUser(undefined);
    setEditTier(undefined);
    if (saved) {
      users.refetch();
      tiers.refetch();
    }
  }

  return (
    <div className="stasharr-users">
      {editUser && (
        <UserDialog user={editUser} tiers={tierList} onClose={refresh} />
      )}
      {editTier && (
        <TierDialog
          tier={editTier}
          libraryPaths={libraryPaths}
          onClose={refresh}
        />
      )}
      {apiKeyUser && (
        <ApiKeyDialog
          user={apiKeyUser}
          onClose={() => {
            setApiKeyUser(undefined);
            users.refetch();
          }}
        />
      )}

      <div className="setting-section">
        <h1>Users</h1>
        <div className="sub-heading">
          The owner account ({me?.username}) from Settings → Security always
          has full access and is not listed here.
        </div>
        <Table className="stasharr-table" responsive>
          <thead>
            <tr>
              <th>User</th>
              <th>Role</th>
              <th>Tier</th>
              <th>Last sign in</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {userList.map((u) => (
              <tr key={u.id} className={u.disabled ? "text-muted" : ""}>
                <td>
                  {u.username}{" "}
                  {u.disabled && <Badge variant="secondary">disabled</Badge>}
                </td>
                <td>{roleLabels[u.role]}</td>
                <td>
                  {u.role === GQL.UserRole.Admin
                    ? "Whole library"
                    : (u.tier?.name ?? "Whole library")}
                </td>
                <td>{formatDate(u.last_login_at)}</td>
                <td className="text-right text-nowrap">
                  <Button
                    size="sm"
                    variant="secondary"
                    title="API key"
                    onClick={() => setApiKeyUser(u)}
                  >
                    <Icon icon={faKey} />
                  </Button>{" "}
                  <Button
                    size="sm"
                    variant="secondary"
                    title="Edit"
                    onClick={() =>
                      setEditUser({
                        id: u.id,
                        username: u.username,
                        password: "",
                        role: u.role,
                        tier_id: u.tier?.id ?? "",
                        disabled: u.disabled,
                      })
                    }
                  >
                    <Icon icon={faPencilAlt} />
                  </Button>{" "}
                  <Button
                    size="sm"
                    variant="danger"
                    title="Delete"
                    onClick={() => onDeleteUser(u)}
                  >
                    <Icon icon={faTrashAlt} />
                  </Button>
                </td>
              </tr>
            ))}
            {userList.length === 0 && (
              <tr>
                <td colSpan={5} className="text-muted">
                  No users yet.
                </td>
              </tr>
            )}
          </tbody>
        </Table>
        <Button onClick={() => setEditUser({ ...emptyUser })}>
          <Icon icon={faPlus} /> Add user
        </Button>
      </div>

      <div className="setting-section">
        <h1>Tiers</h1>
        <div className="sub-heading">
          A tier decides which part of the library its users can see: certain
          folders, without certain tags, and whether they can download or watch
          reels.
        </div>
        <Table className="stasharr-table" responsive>
          <thead>
            <tr>
              <th>Tier</th>
              <th>Library</th>
              <th>Hidden tags</th>
              <th>Users</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {tierList.map((t) => (
              <tr key={t.id}>
                <td>
                  <strong>{t.name}</strong>
                  {t.description && (
                    <div className="text-muted small">{t.description}</div>
                  )}
                </td>
                <td>
                  {t.allowed_paths.length === 0
                    ? "Everything"
                    : t.allowed_paths.map((p) => (
                        <div key={p}>
                          <code>{p}</code>
                        </div>
                      ))}
                  {t.required_tags.length > 0 && (
                    <div className="small">
                      only tagged:{" "}
                      {t.required_tags.map((tag) => tag.name).join(", ")}
                    </div>
                  )}
                </td>
                <td>
                  {t.blocked_tags.length === 0
                    ? "None"
                    : t.blocked_tags.map((tag) => tag.name).join(", ")}
                </td>
                <td>{t.user_count}</td>
                <td className="text-right text-nowrap">
                  <Button
                    size="sm"
                    variant="secondary"
                    title="Edit"
                    onClick={() =>
                      setEditTier({
                        id: t.id,
                        name: t.name,
                        description: t.description ?? "",
                        allowed_paths: t.allowed_paths.join("\n"),
                        blocked_tag_ids: t.blocked_tags.map((x) => x.id),
                        required_tag_ids: t.required_tags.map((x) => x.id),
                        can_download: t.can_download,
                        can_view_reels: t.can_view_reels,
                      })
                    }
                  >
                    <Icon icon={faPencilAlt} />
                  </Button>{" "}
                  <Button
                    size="sm"
                    variant="danger"
                    title="Delete"
                    onClick={() => onDeleteTier(t)}
                  >
                    <Icon icon={faTrashAlt} />
                  </Button>
                </td>
              </tr>
            ))}
            {tierList.length === 0 && (
              <tr>
                <td colSpan={5} className="text-muted">
                  No tiers yet. Users without a tier see the whole library.
                </td>
              </tr>
            )}
          </tbody>
        </Table>
        <Button onClick={() => setEditTier({ ...emptyTier })}>
          <Icon icon={faPlus} /> Add tier
        </Button>
      </div>
    </div>
  );
};

export default UsersPanel;
