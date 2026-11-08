import React, { useState } from "react";
import { Button, Form } from "react-bootstrap";
import { Link, useHistory, useLocation } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { baseName } from "./util";

function clock(sec: number): string {
  const s = Math.max(0, Math.round(sec));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const r = String(s % 60).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${r}` : `${m}:${r}`;
}

const EXAMPLES = ["blonde in red lingerie on a couch", "outdoors by a pool", "kissing in the shower", "close-up of her face"];

// Search by description: type what you want to see, get the scenes and the
// moment in each that looks most like it.
const Describe: React.FC = () => {
  const location = useLocation();
  const history = useHistory();
  const q = new URLSearchParams(location.search).get("q") ?? "";
  const [text, setText] = useState(q);
  const { data, loading, error } = GQL.useSearchMomentsQuery({
    variables: { query: q, limit: 48 },
    skip: q.trim().length < 3,
  });

  function go(v: string) {
    setText(v);
    history.replace({ search: v ? `?q=${encodeURIComponent(v)}` : "" });
  }

  const list = data?.searchMoments ?? [];
  return (
    <div className="stasharr-describe">
      <h2>Search by description</h2>
      <p className="text-muted">
        Describe what you want to see. The Stasharr Tagger looks for it in every indexed scene and shows the moment that
        matches best.
      </p>
      <Form
        onSubmit={(e) => {
          e.preventDefault();
          go(text.trim());
        }}
        className="stasharr-describe-form"
      >
        <Form.Control
          className="text-input"
          autoFocus
          value={text}
          placeholder="e.g. blonde in red lingerie on a couch"
          onChange={(e) => setText(e.currentTarget.value)}
        />
        <Button type="submit" variant="primary" disabled={text.trim().length < 3}>
          Search
        </Button>
      </Form>
      {!q && (
        <div className="stasharr-describe-examples">
          {EXAMPLES.map((x) => (
            <Button key={x} size="sm" variant="secondary" onClick={() => go(x)}>
              {x}
            </Button>
          ))}
        </div>
      )}
      {loading && <LoadingIndicator />}
      {error && <p className="text-danger">{error.message}</p>}
      {q && !loading && !error && list.length === 0 && (
        <p className="text-muted">Nothing found. If scenes are not indexed yet, see Settings → AI markers → Search by description.</p>
      )}
      <div className="stasharr-describe-grid">
        {list.map((m) => (
          <Link key={m.scene.id} to={`/scenes/${m.scene.id}?t=${Math.floor(m.seconds)}`} className="stasharr-describe-item">
            <div className="pic">
              <img src={m.frame} alt="" loading="lazy" />
              <span>▶ {clock(m.seconds)}</span>
            </div>
            <div className="title">{m.scene.title || baseName(m.scene.files[0]?.path ?? "")}</div>
          </Link>
        ))}
      </div>
    </div>
  );
};

export default Describe;

// Settings → AI markers: build the index.
export const SearchIndexSection: React.FC<{ onToast: (msg: string) => void; onError: (e: unknown) => void }> = ({
  onToast,
  onError,
}) => {
  const { data, refetch } = GQL.useSearchIndexStatusQuery({ fetchPolicy: "network-only" });
  const [run] = GQL.useRunSearchIndexMutation();
  const st = data?.searchIndexStatus;
  return (
    <div className="setting-section">
      <h1>Search by description</h1>
      <div className="sub-heading">
        Lets everyone search by describing what they want to see (on the site's search page, and under Describe in
        the menu). The tagger looks at one frame every 10 seconds of each scene once; new scenes are added after
        auto-import. Uses about 1 MB per 100 scenes.
      </div>
      <div className="stasharr-ai-status">
        {st && !st.ready && <span className="text-muted">Needs the Stasharr Tagger to be Ready.</span>}
        {st?.ready && (
          <span>
            {st.done.toLocaleString()} of {st.total.toLocaleString()} scenes indexed ({st.frames.toLocaleString()} frames)
          </span>
        )}
        <Button
          size="sm"
          variant="primary"
          disabled={!st?.ready || st.done >= st.total}
          onClick={async () => {
            try {
              await run({ variables: {} });
              onToast("Indexing scenes. See Settings → Tasks for progress.");
              window.setTimeout(() => refetch(), 5000);
            } catch (e) {
              onError(e);
            }
          }}
        >
          Index the rest
        </Button>
        <Link to="/describe" className="btn btn-sm btn-secondary">
          Try it
        </Link>
      </div>
    </div>
  );
};
