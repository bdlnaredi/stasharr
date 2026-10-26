import React, { useMemo, useState } from "react";
import { Button, Card, Form } from "react-bootstrap";
import { Link, Route, Switch, useHistory, useParams } from "react-router-dom";
import { Helmet } from "react-helmet";
import {
  faArrowDown,
  faArrowUp,
  faPen,
  faPlay,
  faPlus,
  faRandom,
  faTimes,
  faTrashAlt,
} from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { SceneQueue } from "src/models/sceneQueue";
import { SceneCard } from "src/components/Scenes/SceneCard";
import { NewPlaylistModal } from "./PlaylistButtons";
import { formatDuration, sceneName } from "./util";

const refetchPlaylists = ["FindPlaylists", "FindPlaylist", "ScenePlaylists"];

const PlaylistCard: React.FC<{ playlist: GQL.PlaylistDataFragment }> = ({
  playlist,
}) => (
  <Link to={`/playlists/${playlist.id}`} className="stasharr-playlist-card">
    <Card>
      <div className="stasharr-playlist-cover">
        {playlist.cover?.paths.screenshot ? (
          <img src={playlist.cover.paths.screenshot} alt="" loading="lazy" />
        ) : (
          <div className="stasharr-playlist-empty">Empty</div>
        )}
        <span className="stasharr-playlist-count">
          {playlist.scene_count} scene{playlist.scene_count === 1 ? "" : "s"}
        </span>
      </div>
      <Card.Body>
        <h5>
          {playlist.name}
          {playlist.smart && <span className="stasharr-smart-badge">Smart</span>}
        </h5>
        {playlist.description && (
          <div className="text-muted stasharr-playlist-desc">
            {playlist.description}
          </div>
        )}
      </Card.Body>
    </Card>
  </Link>
);

const ContinueWatchingGrid: React.FC = () => {
  const { data } = GQL.useContinueWatchingQuery({
    variables: { limit: 12 },
    fetchPolicy: "cache-and-network",
  });
  const scenes = data?.continueWatching ?? [];
  const queue = useMemo(
    () => SceneQueue.fromSceneIDList(scenes.map((s) => s.id)),
    [scenes]
  );
  if (scenes.length === 0) return null;
  return (
    <div className="stasharr-playlists-section">
      <h3>Continue watching</h3>
      <div className="row justify-content-center">
        {scenes.map((s, i) => (
          <SceneCard key={s.id} scene={s} queue={queue} index={i} zoomIndex={1} />
        ))}
      </div>
    </div>
  );
};

const PlaylistList: React.FC = () => {
  const { data, loading } = GQL.useFindPlaylistsQuery({
    fetchPolicy: "cache-and-network",
  });
  const [showNew, setShowNew] = useState(false);
  const history = useHistory();

  const playlists = data?.findPlaylists ?? [];

  return (
    <div className="stasharr-playlists">
      <Helmet>
        <title>Playlists</title>
      </Helmet>
      <ContinueWatchingGrid />

      <div className="stasharr-playlists-section">
        <div className="stasharr-section-head">
          <h3>Playlists</h3>
          <Button variant="primary" onClick={() => setShowNew(true)}>
            <Icon icon={faPlus} /> New playlist
          </Button>
        </div>
        {loading && !data ? (
          <LoadingIndicator />
        ) : (
          <div className="stasharr-playlist-grid">
            {playlists.map((p) => (
              <PlaylistCard key={p.id} playlist={p} />
            ))}
          </div>
        )}
      </div>

      {showNew && (
        <NewPlaylistModal
          sceneIDs={[]}
          onClose={(created) => {
            setShowNew(false);
            if (created) history.push(`/playlists/${created.id}`);
          }}
        />
      )}
    </div>
  );
};

const PlaylistDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const history = useHistory();
  const Toast = useToast();
  const { data, loading } = GQL.useFindPlaylistQuery({
    variables: { id },
    fetchPolicy: "cache-and-network",
  });
  const [update] = GQL.usePlaylistUpdateMutation({
    refetchQueries: refetchPlaylists,
  });
  const [destroy] = GQL.usePlaylistDestroyMutation({
    refetchQueries: ["FindPlaylists", "ScenePlaylists"],
  });
  const [removeScenes] = GQL.usePlaylistRemoveScenesMutation({
    refetchQueries: refetchPlaylists,
  });
  const [reorder] = GQL.usePlaylistReorderMutation({
    refetchQueries: refetchPlaylists,
  });
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const playlist = data?.findPlaylist;
  const scenes = useMemo(() => playlist?.scenes ?? [], [playlist]);
  const ids = scenes.map((s) => s.id);

  if (loading && !data) return <LoadingIndicator />;
  if (!playlist) {
    return (
      <div className="stasharr-playlists">
        <h3>Playlist not found</h3>
        <Link to="/playlists">Back to playlists</Link>
      </div>
    );
  }
  const p = playlist;

  function play(shuffle: boolean) {
    if (ids.length === 0) return;
    const order = [...ids];
    if (shuffle) {
      for (let i = order.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [order[i], order[j]] = [order[j], order[i]];
      }
    }
    const queue = SceneQueue.fromSceneIDList(order);
    history.push(queue.makeLink(order[0], { sceneIndex: 0, continue: true, autoPlay: true }));
  }

  function playFrom(index: number) {
    const queue = SceneQueue.fromSceneIDList(ids);
    history.push(queue.makeLink(ids[index], { sceneIndex: index, continue: true, autoPlay: true }));
  }

  async function move(index: number, delta: number) {
    const to = index + delta;
    if (to < 0 || to >= ids.length) return;
    const order = [...ids];
    [order[index], order[to]] = [order[to], order[index]];
    try {
      await reorder({ variables: { id: p.id, scene_ids: order } });
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onRemove(sceneID: string) {
    try {
      await removeScenes({ variables: { id: p.id, scene_ids: [sceneID] } });
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onSave() {
    try {
      await update({
        variables: {
          input: {
            id: p.id,
            name: p.watch_later ? undefined : name,
            description,
          },
        },
      });
      setEditing(false);
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onDelete() {
    if (!window.confirm(`Delete the playlist ${p.name}? The scenes are not deleted.`)) {
      return;
    }
    try {
      await destroy({ variables: { id: p.id } });
      history.push("/playlists");
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div className="stasharr-playlists">
      <Helmet>
        <title>{p.name}</title>
      </Helmet>
      <div className="stasharr-playlist-header">
        <div>
          <Link to="/playlists" className="text-muted">
            Playlists
          </Link>
          {editing ? (
            <Form
              onSubmit={(e) => {
                e.preventDefault();
                onSave();
              }}
            >
              {!p.watch_later && (
                <Form.Control
                  className="text-input mb-2"
                  value={name}
                  onChange={(e) => setName(e.currentTarget.value)}
                />
              )}
              <Form.Control
                className="text-input mb-2"
                placeholder="Description"
                value={description}
                onChange={(e) => setDescription(e.currentTarget.value)}
              />
              <Button type="submit" variant="primary" size="sm">
                Save
              </Button>{" "}
              <Button variant="secondary" size="sm" onClick={() => setEditing(false)}>
                Cancel
              </Button>
            </Form>
          ) : (
            <>
              <h2>
                {p.name}
                {p.smart && <span className="stasharr-smart-badge">Smart</span>}
              </h2>
              {p.description && <div className="text-muted">{p.description}</div>}
              {p.smart && (
                <div className="text-muted">
                  Updates itself from a saved filter. To change the filter,
                  filter the <Link to="/scenes">Scenes</Link> list and choose ⋯ →
                  Save as smart playlist.
                </div>
              )}
              <div className="text-muted">
                {scenes.length} scene{scenes.length === 1 ? "" : "s"}
              </div>
            </>
          )}
        </div>
        {!editing && (
          <div className="stasharr-playlist-actions">
            <Button variant="primary" disabled={!ids.length} onClick={() => play(false)}>
              <Icon icon={faPlay} /> Play all
            </Button>{" "}
            <Button variant="secondary" disabled={ids.length < 2} onClick={() => play(true)}>
              <Icon icon={faRandom} /> Shuffle
            </Button>{" "}
            <Button
              variant="secondary"
              title="Edit"
              onClick={() => {
                setName(p.name);
                setDescription(p.description);
                setEditing(true);
              }}
            >
              <Icon icon={faPen} />
            </Button>{" "}
            {!p.watch_later && (
              <Button variant="danger" title="Delete playlist" onClick={() => onDelete()}>
                <Icon icon={faTrashAlt} />
              </Button>
            )}
          </div>
        )}
      </div>

      {scenes.length === 0 ? (
        <div className="text-muted stasharr-playlists-section">
          {p.smart
            ? "No scenes match this playlist's filter right now."
            : p.watch_later
            ? "Nothing here yet. Use the bookmark button on a scene to save it for later."
            : "Nothing here yet. Add scenes from a scene's playlist button, or select scenes in the Scenes list and choose Add to playlist."}
        </div>
      ) : (
        <div className="stasharr-playlist-items">
          {scenes.map((s, i) => (
            <div className="stasharr-playlist-item" key={s.id}>
              <span className="stasharr-playlist-index">{i + 1}</span>
              <button
                type="button"
                className="stasharr-playlist-thumb"
                onClick={() => playFrom(i)}
                title="Play from here"
              >
                {s.paths.screenshot && <img src={s.paths.screenshot} alt="" loading="lazy" />}
                <span>{formatDuration(s.files[0]?.duration)}</span>
              </button>
              <div className="stasharr-playlist-title">
                <Link to={`/scenes/${s.id}`}>{sceneName(s)}</Link>
                <div className="text-muted">
                  {[s.studio?.name, s.date].filter(Boolean).join(" · ")}
                </div>
              </div>
              {!p.smart && (
              <div className="stasharr-playlist-item-actions">
                <Button variant="link" disabled={i === 0} title="Move up" onClick={() => move(i, -1)}>
                  <Icon icon={faArrowUp} />
                </Button>
                <Button
                  variant="link"
                  disabled={i === scenes.length - 1}
                  title="Move down"
                  onClick={() => move(i, 1)}
                >
                  <Icon icon={faArrowDown} />
                </Button>
                <Button variant="link" title="Remove from playlist" onClick={() => onRemove(s.id)}>
                  <Icon icon={faTimes} />
                </Button>
              </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const Playlists: React.FC = () => (
  <Switch>
    <Route exact path="/playlists" component={PlaylistList} />
    <Route path="/playlists/:id" component={PlaylistDetails} />
  </Switch>
);

export default Playlists;
