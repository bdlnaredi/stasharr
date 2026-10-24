import React, { useMemo } from "react";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { SceneQueue } from "src/models/sceneQueue";
import { formatDuration, sceneName } from "./util";

// "More like this" for the scene page: scenes sharing performers, studio
// and tags, best match first.
export const SimilarScenesPanel: React.FC<{ sceneId: string }> = ({
  sceneId,
}) => {
  const { data, loading } = GQL.useFindSimilarScenesQuery({
    variables: { scene_id: sceneId, limit: 24 },
    fetchPolicy: "cache-and-network",
  });
  const scenes = useMemo(() => data?.findSimilarScenes ?? [], [data]);
  const queue = useMemo(
    () => SceneQueue.fromSceneIDList(scenes.map((s) => s.id)),
    [scenes]
  );

  if (loading && !data) return <LoadingIndicator />;
  if (scenes.length === 0) {
    return (
      <div className="text-muted p-2">
        Nothing similar yet. Scenes are matched on performers, studio and tags,
        so identify this scene first.
      </div>
    );
  }

  return (
    <div className="stasharr-similar">
      {scenes.map((s, i) => (
        <Link
          key={s.id}
          className="stasharr-similar-item"
          to={queue.makeLink(s.id, { sceneIndex: i, continue: true })}
        >
          <span className="stasharr-playlist-thumb">
            {s.paths.screenshot && (
              <img src={s.paths.screenshot} alt="" loading="lazy" />
            )}
            <span>{formatDuration(s.files[0]?.duration)}</span>
          </span>
          <span className="stasharr-similar-text">
            <span className="stasharr-similar-title">{sceneName(s)}</span>
            <small className="text-muted">
              {[s.studio?.name, s.performers.map((p) => p.name).slice(0, 3).join(", ")]
                .filter(Boolean)
                .join(" · ")}
            </small>
          </span>
        </Link>
      ))}
    </div>
  );
};

export default SimilarScenesPanel;
