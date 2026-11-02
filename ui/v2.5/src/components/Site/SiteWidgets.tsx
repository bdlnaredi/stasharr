import React, { useEffect, useState } from "react";
import * as GQL from "src/core/generated-graphql";
import { count, t, views } from "./i18n";

// A heart that likes or unlikes a scene for the current user.
export const LikeButton: React.FC<{ sceneID: string; className?: string; label?: boolean }> = ({
  sceneID,
  className,
  label = true,
}) => {
  const { data } = GQL.useSiteSceneExtrasQuery({ variables: { id: sceneID }, fetchPolicy: "cache-first" });
  const [liked, setLiked] = useState(false);
  const [toggle, { loading }] = GQL.useSiteToggleLikeMutation({
    refetchQueries: ["SiteLiked"],
  });
  useEffect(() => setLiked(!!data?.findScene?.site_liked), [data]);

  return (
    <button
      type="button"
      className={`site-btn site-btn-ghost site-like ${liked ? "on" : ""} ${className ?? ""}`}
      disabled={loading}
      aria-pressed={liked}
      title={liked ? t("Unlike") : t("Like")}
      onClick={async () => {
        setLiked(!liked);
        try {
          const res = await toggle({ variables: { scene_id: sceneID } });
          setLiked(!!res.data?.siteToggleLike);
        } catch {
          setLiked(liked);
        }
      }}
    >
      <span className="site-heart">{liked ? "♥" : "♡"}</span>
      {label && ` ${liked ? t("Liked") : t("Like")}`}
    </button>
  );
};

// Follows or unfollows a model.
export const FollowButton: React.FC<{ performerID: string; following: boolean }> = ({
  performerID,
  following,
}) => {
  const [on, setOn] = useState(following);
  useEffect(() => setOn(following), [following]);
  const [toggle, { loading }] = GQL.useSiteToggleFavoriteModelMutation({
    refetchQueries: ["SiteFavoriteModels", "SiteFromFavorites"],
  });
  return (
    <button
      type="button"
      className={`site-btn ${on ? "site-btn-ghost" : "site-btn-primary"} site-like ${on ? "on" : ""}`}
      disabled={loading}
      aria-pressed={on}
      onClick={async () => {
        setOn(!on);
        try {
          const res = await toggle({ variables: { performer_id: performerID } });
          setOn(!!res.data?.siteToggleFavoriteModel);
        } catch {
          setOn(on);
        }
      }}
    >
      <span className="site-heart">{on ? "♥" : "♡"}</span> {on ? t("Favourite") : t("Add to favourites")}
    </button>
  );
};

// "1.2K views" for a scene, counted across everyone.
export const SceneViews: React.FC<{ sceneID: string }> = ({ sceneID }) => {
  const { data } = GQL.useSiteSceneExtrasQuery({ variables: { id: sceneID }, fetchPolicy: "cache-and-network" });
  const n = data?.findScene?.site_views;
  if (n === undefined) return null;
  return <span className="site-views">👁 {views(n)}</span>;
};

// Five stars: the average of everyone's ratings, and the current user's own
// rating, which they can change by clicking.
export const SceneRating: React.FC<{ sceneID: string }> = ({ sceneID }) => {
  const { data, refetch } = GQL.useSiteSceneExtrasQuery({
    variables: { id: sceneID },
    fetchPolicy: "cache-and-network",
  });
  const [rate] = GQL.useSiteRateSceneMutation();
  const [hover, setHover] = useState(0);
  const mine = data?.findScene?.rating100 ?? null;
  const [own, setOwn] = useState<number | null>(mine);
  useEffect(() => setOwn(mine), [mine]);
  const summary = data?.findScene?.site_rating;
  const avg = summary?.average ?? null;
  const ownStars = own ? Math.round(own / 20) : 0;

  async function set(stars: number) {
    const value = stars === ownStars ? null : stars * 20;
    setOwn(value);
    try {
      await rate({ variables: { id: sceneID, rating100: value } });
      refetch();
    } catch {
      setOwn(mine);
    }
  }

  return (
    <div className="site-rating">
      {avg !== null && summary && (
        <div className="site-rating-avg" title={count(summary.count, "rating")}>
          <span className="site-stars" style={{ ["--fill" as string]: `${avg}%` }}>
            ★★★★★
          </span>
          <b>{(avg / 20).toFixed(1)}</b>
          <small>
            {count(summary.count, "rating")}
          </small>
        </div>
      )}
      <div className="site-rating-own" onMouseLeave={() => setHover(0)}>
        <small>{ownStars ? t("Your rating") : t("Rate it")}</small>
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            type="button"
            key={n}
            className={(hover || ownStars) >= n ? "on" : ""}
            onMouseEnter={() => setHover(n)}
            onClick={() => set(n)}
            aria-label={`${n} ★`}
          >
            ★
          </button>
        ))}
      </div>
    </div>
  );
};
