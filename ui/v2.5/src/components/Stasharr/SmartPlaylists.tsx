import React, { useMemo, useState } from "react";
import { Button, Form, Modal } from "react-bootstrap";
import { Link } from "react-router-dom";
import Slider from "@ant-design/react-slick";
import * as GQL from "src/core/generated-graphql";
import { getSlickSliderSettings } from "src/core/recommendations";
import { ListFilterModel } from "src/models/list-filter/filter";
import { SceneQueue } from "src/models/sceneQueue";
import { SceneCard } from "src/components/Scenes/SceneCard";
import { RecommendationRow } from "src/components/FrontPage/RecommendationRow";
import { useToast } from "src/hooks/Toast";

const refetch = ["FindPlaylists", "FindPlaylist", "SmartPlaylistRows"];

function ruleFromFilter(
  filter: ListFilterModel,
  limit: number
): GQL.SmartPlaylistRuleInput {
  const find = filter.makeFindFilter();
  return {
    scene_filter: filter.makeFilter() as GQL.SceneFilterType,
    sort: find.sort,
    direction: find.direction,
    limit,
  };
}

// Saves the Scenes list's current filter and sort as a smart playlist, or
// replaces the filter of an existing one.
export const SmartPlaylistModal: React.FC<{
  filter: ListFilterModel;
  onClose: () => void;
}> = ({ filter, onClose }) => {
  const Toast = useToast();
  const { data } = GQL.useFindPlaylistsQuery({ fetchPolicy: "network-only" });
  const smart = (data?.findPlaylists ?? []).filter((p) => p.smart);
  const [target, setTarget] = useState("new");
  const [name, setName] = useState("");
  const [limit, setLimit] = useState(100);
  const [create, { loading: creating }] = GQL.usePlaylistCreateMutation({
    refetchQueries: refetch,
  });
  const [update, { loading: updating }] = GQL.usePlaylistUpdateMutation({
    refetchQueries: refetch,
  });

  const criteria = filter.criteria.length;

  async function onSave() {
    const smart_rule = ruleFromFilter(filter, limit);
    try {
      if (target === "new") {
        await create({ variables: { input: { name, smart_rule } } });
        Toast.success(`Created smart playlist ${name.trim()}`);
      } else {
        await update({ variables: { input: { id: target, smart_rule } } });
        Toast.success("Smart playlist updated");
      }
      onClose();
    } catch (e) {
      Toast.error(e);
    }
  }

  const canSave = target !== "new" || name.trim() !== "";

  return (
    <Modal show onHide={() => onClose()}>
      <Modal.Header closeButton>Save as smart playlist</Modal.Header>
      <Modal.Body>
        <p className="text-muted">
          A smart playlist keeps itself up to date: it always holds the scenes
          that match this list&apos;s current filters ({criteria} filter
          {criteria === 1 ? "" : "s"}) and sort order. It shows as a row on the
          home page and in the app.
        </p>
        <Form
          onSubmit={(e) => {
            e.preventDefault();
            if (canSave) onSave();
          }}
        >
          {smart.length > 0 && (
            <Form.Group>
              <Form.Label>Save to</Form.Label>
              <Form.Control
                as="select"
                className="input-control"
                value={target}
                onChange={(e) => setTarget(e.currentTarget.value)}
              >
                <option value="new">A new smart playlist</option>
                {smart.map((p) => (
                  <option key={p.id} value={p.id}>
                    Replace the filter of {p.name}
                  </option>
                ))}
              </Form.Control>
            </Form.Group>
          )}
          {target === "new" && (
            <Form.Group>
              <Form.Label>Name</Form.Label>
              <Form.Control
                className="text-input"
                autoFocus
                placeholder="For example: Unwatched 4K"
                value={name}
                onChange={(e) => setName(e.currentTarget.value)}
              />
            </Form.Group>
          )}
          <Form.Group>
            <Form.Label>Most scenes to hold</Form.Label>
            <Form.Control
              className="text-input"
              type="number"
              min={1}
              max={1000}
              value={limit}
              onChange={(e) =>
                setLimit(Math.max(1, Math.min(1000, Number(e.currentTarget.value) || 100)))
              }
            />
          </Form.Group>
        </Form>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={() => onClose()}>
          Cancel
        </Button>
        <Button
          variant="primary"
          disabled={creating || updating || !canSave}
          onClick={() => onSave()}
        >
          Save
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

function isTouchEnabled() {
  return "ontouchstart" in window || navigator.maxTouchPoints > 0;
}

const SmartRow: React.FC<{
  id: string;
  name: string;
  scenes: GQL.SlimSceneDataFragment[];
}> = ({ id, name, scenes }) => {
  const queue = useMemo(
    () => SceneQueue.fromSceneIDList(scenes.map((s) => s.id)),
    [scenes]
  );
  if (scenes.length === 0) return null;
  return (
    <RecommendationRow
      className="scene-recommendations stasharr-recommended"
      header={name}
      link={<Link to={`/playlists/${id}`}>See all</Link>}
    >
      <Slider {...getSlickSliderSettings(scenes.length, isTouchEnabled())}>
        {scenes.map((scene, index) => (
          <SceneCard
            key={scene.id}
            scene={scene}
            queue={queue}
            index={index}
            zoomIndex={1}
          />
        ))}
      </Slider>
    </RecommendationRow>
  );
};

// One front page row per smart playlist.
export const SmartPlaylistRows: React.FC = () => {
  const { data } = GQL.useSmartPlaylistRowsQuery({
    variables: { limit: 24 },
    fetchPolicy: "cache-and-network",
  });
  const rows = (data?.findPlaylists ?? []).filter((p) => p.smart);
  return (
    <>
      {rows.map((p) => (
        <SmartRow key={p.id} id={p.id} name={p.name} scenes={p.scenes} />
      ))}
    </>
  );
};
