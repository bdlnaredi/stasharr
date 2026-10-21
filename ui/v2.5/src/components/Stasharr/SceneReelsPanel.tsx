import React, { useEffect } from "react";
import { Button } from "react-bootstrap";
import { faTrashAlt } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { useCurrentUser } from "./CurrentUser";

function formatTime(seconds: number) {
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export const SceneReelsPanel: React.FC<{
  sceneId: string;
  isVisible: boolean;
}> = ({ sceneId, isVisible }) => {
  const Toast = useToast();
  const { isEditor } = useCurrentUser();
  const { data, loading, refetch } = GQL.useSceneReelsQuery({
    variables: { id: sceneId },
    fetchPolicy: "network-only",
  });
  const [generateReels] = GQL.useGenerateReelsMutation();
  const [destroyReel] = GQL.useReelDestroyMutation();

  // pick up reels finished while the tab was closed
  useEffect(() => {
    if (isVisible) refetch();
  }, [isVisible, refetch]);

  if (loading) return <LoadingIndicator />;
  const reels = data?.findScene?.reels ?? [];

  async function make() {
    try {
      await generateReels({
        variables: { input: { scene_ids: [sceneId], overwrite: true } },
      });
      Toast.success("Making a reel. Open this tab again in a minute to see it.");
    } catch (e) {
      Toast.error(e);
    }
  }

  async function remove(id: string) {
    try {
      await destroyReel({ variables: { id } });
      refetch();
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div className="stasharr-scene-reels">
      {reels.length === 0 && (
        <p className="text-muted">This scene has no reel yet.</p>
      )}
      {reels.map((r) => (
        <div key={r.id} className="stasharr-scene-reel">
          <video src={r.stream} controls loop playsInline preload="metadata" />
          <div className="stasharr-scene-reel-info">
            <div>
              {Math.round(r.duration)}s reel, moments at{" "}
              {r.segments.map((s) => formatTime(s.start)).join(", ")}
            </div>
            {isEditor && (
              <Button
                size="sm"
                variant="danger"
                title="Delete reel"
                onClick={() => remove(r.id)}
              >
                <Icon icon={faTrashAlt} />
              </Button>
            )}
          </div>
        </div>
      ))}
      {isEditor && (
        <Button variant="secondary" onClick={() => make()}>
          {reels.length > 0 ? "Make a new reel" : "Make reel"}
        </Button>
      )}
    </div>
  );
};

export default SceneReelsPanel;
