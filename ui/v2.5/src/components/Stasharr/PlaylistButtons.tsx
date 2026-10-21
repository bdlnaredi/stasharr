import React, { useState } from "react";
import { Button, Dropdown, Form, Modal } from "react-bootstrap";
import { faBookmark as faBookmarkSolid, faCheck, faListUl, faPlus } from "@fortawesome/free-solid-svg-icons";
import { faBookmark } from "@fortawesome/free-regular-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { useToast } from "src/hooks/Toast";

const refetchPlaylists = ["FindPlaylists", "FindPlaylist", "ScenePlaylists"];

// Asks for a name and creates a playlist with the given scenes.
export const NewPlaylistModal: React.FC<{
  sceneIDs: string[];
  onClose: (created?: GQL.PlaylistDataFragment) => void;
}> = ({ sceneIDs, onClose }) => {
  const Toast = useToast();
  const [name, setName] = useState("");
  const [create, { loading }] = GQL.usePlaylistCreateMutation({
    refetchQueries: refetchPlaylists,
  });

  async function onSave() {
    try {
      const res = await create({
        variables: { input: { name, scene_ids: sceneIDs } },
      });
      const p = res.data?.playlistCreate;
      if (p) {
        Toast.success(
          sceneIDs.length
            ? `Added ${sceneIDs.length} scene(s) to ${p.name}`
            : `Created ${p.name}`
        );
      }
      onClose(p ?? undefined);
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <Modal show onHide={() => onClose()}>
      <Modal.Header closeButton>New playlist</Modal.Header>
      <Modal.Body>
        <Form
          onSubmit={(e) => {
            e.preventDefault();
            if (name.trim()) onSave();
          }}
        >
          <Form.Control
            className="text-input"
            autoFocus
            placeholder="Playlist name"
            value={name}
            onChange={(e) => setName(e.currentTarget.value)}
          />
        </Form>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={() => onClose()}>
          Cancel
        </Button>
        <Button
          variant="primary"
          disabled={loading || !name.trim()}
          onClick={() => onSave()}
        >
          Create
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

// Watch later toggle and "add to playlist" menu for the scene page.
export const PlaylistButtons: React.FC<{ sceneID: string }> = ({ sceneID }) => {
  const Toast = useToast();
  const { data } = GQL.useScenePlaylistsQuery({
    variables: { id: sceneID },
    fetchPolicy: "cache-and-network",
  });
  const { data: playlistsData } = GQL.useFindPlaylistsQuery({
    fetchPolicy: "cache-and-network",
  });
  const [toggle] = GQL.useWatchLaterToggleMutation({
    refetchQueries: refetchPlaylists,
  });
  const [add] = GQL.usePlaylistAddScenesMutation({
    refetchQueries: refetchPlaylists,
  });
  const [remove] = GQL.usePlaylistRemoveScenesMutation({
    refetchQueries: refetchPlaylists,
  });
  const [showNew, setShowNew] = useState(false);

  const inWatchLater = data?.findScene?.in_watch_later ?? false;
  const inPlaylists = new Set(data?.findScene?.playlist_ids ?? []);
  const playlists = (playlistsData?.findPlaylists ?? []).filter(
    (p) => !p.watch_later
  );

  async function onToggleWatchLater() {
    try {
      const res = await toggle({ variables: { scene_id: sceneID } });
      Toast.success(
        res.data?.watchLaterToggle
          ? "Added to Watch later"
          : "Removed from Watch later"
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onTogglePlaylist(p: GQL.PlaylistDataFragment) {
    try {
      if (inPlaylists.has(p.id)) {
        await remove({ variables: { id: p.id, scene_ids: [sceneID] } });
        Toast.success(`Removed from ${p.name}`);
      } else {
        await add({ variables: { id: p.id, scene_ids: [sceneID] } });
        Toast.success(`Added to ${p.name}`);
      }
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <>
      <span>
        <Button
          className="minimal stasharr-watch-later"
          variant="secondary"
          title={inWatchLater ? "Remove from Watch later" : "Watch later"}
          onClick={() => onToggleWatchLater()}
        >
          <Icon icon={inWatchLater ? faBookmarkSolid : faBookmark} />
        </Button>
      </span>
      <span>
        <Dropdown>
          <Dropdown.Toggle
            variant="secondary"
            className="minimal"
            id="add-to-playlist"
            title="Add to playlist"
          >
            <Icon icon={faListUl} />
          </Dropdown.Toggle>
          <Dropdown.Menu className="bg-secondary text-white">
            {playlists.map((p) => (
              <Dropdown.Item
                key={p.id}
                className="bg-secondary text-white"
                onClick={() => onTogglePlaylist(p)}
              >
                <span className="stasharr-check">
                  {inPlaylists.has(p.id) && <Icon icon={faCheck} />}
                </span>
                {p.name}
              </Dropdown.Item>
            ))}
            {playlists.length > 0 && <Dropdown.Divider />}
            <Dropdown.Item
              className="bg-secondary text-white"
              onClick={() => setShowNew(true)}
            >
              <span className="stasharr-check">
                <Icon icon={faPlus} />
              </span>
              New playlist…
            </Dropdown.Item>
          </Dropdown.Menu>
        </Dropdown>
      </span>
      {showNew && (
        <NewPlaylistModal sceneIDs={[sceneID]} onClose={() => setShowNew(false)} />
      )}
    </>
  );
};

// Adds several scenes to a playlist, for the scene list's ⋯ menu.
export const AddToPlaylistDialog: React.FC<{
  sceneIDs: string[];
  onClose: () => void;
}> = ({ sceneIDs, onClose }) => {
  const Toast = useToast();
  const { data, loading } = GQL.useFindPlaylistsQuery({
    fetchPolicy: "network-only",
  });
  const [add] = GQL.usePlaylistAddScenesMutation({
    refetchQueries: refetchPlaylists,
  });
  const [creating, setCreating] = useState(false);

  if (creating) {
    return <NewPlaylistModal sceneIDs={sceneIDs} onClose={() => onClose()} />;
  }

  async function onPick(p: GQL.PlaylistDataFragment) {
    try {
      await add({ variables: { id: p.id, scene_ids: sceneIDs } });
      Toast.success(`Added ${sceneIDs.length} scene(s) to ${p.name}`);
      onClose();
    } catch (e) {
      Toast.error(e);
    }
  }

  const playlists = data?.findPlaylists ?? [];

  return (
    <Modal show onHide={() => onClose()}>
      <Modal.Header closeButton>
        Add {sceneIDs.length} scene(s) to a playlist
      </Modal.Header>
      <Modal.Body className="stasharr-playlist-picker">
        {loading && !data ? (
          <div className="text-muted">Loading…</div>
        ) : (
          playlists.map((p) => (
            <Button
              key={p.id}
              variant="secondary"
              block
              onClick={() => onPick(p)}
            >
              {p.name}{" "}
              <small className="text-muted">({p.scene_count})</small>
            </Button>
          ))
        )}
        <Button variant="primary" block onClick={() => setCreating(true)}>
          <Icon icon={faPlus} /> New playlist
        </Button>
      </Modal.Body>
    </Modal>
  );
};
