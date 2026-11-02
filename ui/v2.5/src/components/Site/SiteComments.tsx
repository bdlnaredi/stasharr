import React, { useState } from "react";
import * as GQL from "src/core/generated-graphql";
import { ago, count, t } from "./i18n";

const PER_PAGE = 20;

// A scene's comments: newest first, with a box to add one. On a reel the
// same conversation is shown, and new comments note the reel.
export const SiteComments: React.FC<{ sceneID: string; reelID?: string; compact?: boolean }> = ({
  sceneID,
  reelID,
  compact,
}) => {
  const [pages, setPages] = useState(1);
  const [body, setBody] = useState("");
  const [error, setError] = useState("");
  const { data, loading, refetch } = GQL.useSiteCommentsQuery({
    variables: { scene_id: sceneID, page: 1, per_page: PER_PAGE * pages },
    fetchPolicy: "cache-and-network",
  });
  const [add, adding] = GQL.useSiteAddCommentMutation({ refetchQueries: ["SiteSceneExtras"] });
  const [remove] = GQL.useSiteDeleteCommentMutation({ refetchQueries: ["SiteSceneExtras"] });

  const list = data?.siteComments.comments ?? [];
  const total = data?.siteComments.count ?? 0;

  async function post(e: React.FormEvent) {
    e.preventDefault();
    const text = body.trim();
    if (!text) return;
    setError("");
    try {
      await add({ variables: { scene_id: sceneID, body: text, reel_id: reelID } });
      setBody("");
      refetch();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function del(id: string) {
    try {
      await remove({ variables: { id } });
      refetch();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <section className={`site-comments ${compact ? "compact" : ""}`}>
      {!compact && <h2 className="site-comments-title">{total > 0 ? count(total, "comment") : t("Comments")}</h2>}
      <form className="site-comment-form" onSubmit={post}>
        <textarea
          value={body}
          maxLength={1000}
          rows={compact ? 2 : 3}
          placeholder={t("Add a comment…")}
          onChange={(e) => setBody(e.currentTarget.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) post(e);
          }}
        />
        <button type="submit" className="site-btn site-btn-primary" disabled={!body.trim() || adding.loading}>
          {t("Post")}
        </button>
      </form>
      {error && <div className="site-comment-error">{error}</div>}
      {list.length === 0 && !loading && <div className="site-comment-empty">{t("Be the first to comment.")}</div>}
      <ul className="site-comment-list">
        {list.map((c) => (
          <li key={c.id} className={c.mine ? "mine" : ""}>
            <span className="site-comment-avatar">{c.author.slice(0, 1).toUpperCase()}</span>
            <div className="site-comment-main">
              <div className="site-comment-head">
                <b>{c.author}</b>
                <span>{ago(c.created_at)}</span>
                {c.from_reel && <span className="site-comment-tag">{t("on a reel")}</span>}
                {c.can_delete && (
                  <button type="button" className="site-comment-delete" onClick={() => del(c.id)}>
                    {t("Delete")}
                  </button>
                )}
              </div>
              <p>{c.body}</p>
            </div>
          </li>
        ))}
      </ul>
      {list.length < total && (
        <button type="button" className="site-btn site-btn-ghost site-comment-more" onClick={() => setPages(pages + 1)}>
          {t("Show more comments")}
        </button>
      )}
    </section>
  );
};
