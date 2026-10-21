import React, { useMemo } from "react";
import { Link } from "react-router-dom";
import Slider from "@ant-design/react-slick";
import * as GQL from "src/core/generated-graphql";
import { getSlickSliderSettings } from "src/core/recommendations";
import { SceneQueue } from "src/models/sceneQueue";
import { SceneCard } from "src/components/Scenes/SceneCard";
import { RecommendationRow } from "src/components/FrontPage/RecommendationRow";

function isTouchEnabled() {
  return "ontouchstart" in window || navigator.maxTouchPoints > 0;
}

// Scenes the user started but did not finish, for the front page.
export const ContinueWatchingRow: React.FC = () => {
  const { data } = GQL.useContinueWatchingQuery({
    variables: { limit: 20 },
    fetchPolicy: "cache-and-network",
  });
  const scenes = data?.continueWatching ?? [];
  const queue = useMemo(
    () => SceneQueue.fromSceneIDList(scenes.map((s) => s.id)),
    [scenes]
  );

  if (scenes.length === 0) return null;

  return (
    <RecommendationRow
      className="scene-recommendations stasharr-continue-watching"
      header="Continue watching"
      link={<Link to="/playlists">Playlists</Link>}
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

export default ContinueWatchingRow;
