import React from "react";
import { Button } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { SettingRow } from "./util";

// Settings: short looping clips for tags that have no picture of their own.
export const TagClipsSection: React.FC = () => {
  const Toast = useToast();
  const { data, refetch } = GQL.useTagClipProgressQuery({ fetchPolicy: "network-only" });
  const [make] = GQL.useMakeTagClipsMutation();
  const p = data?.tagClipProgress;

  async function run(redo: boolean) {
    if (redo && !window.confirm("Make every tag's clip again? Takes a few seconds per tag.")) return;
    try {
      await make({ variables: { redo } });
      Toast.success("Making tag clips. See Settings → Tasks for progress.");
      window.setTimeout(() => refetch(), 4000);
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div className="setting-section">
      <h1>Tag clips</h1>
      <div className="sub-heading">
        Every tag without its own picture gets a 4-second silent loop from its best scene, starting at a marker for
        the tag when there is one (yours or an AI marker). They play on the site's categories, the Tags pages and
        the app. New tags get one after auto-import. Tags with a picture you set keep it.
      </div>
      <SettingRow heading="Tags with a clip" sub="Make clips does tags that have none yet; Remake all picks again, for example after adding markers.">
        <div className="stasharr-ai-status">
          {p && (
            <span>
              {p.done.toLocaleString()} of {p.total.toLocaleString()}
            </span>
          )}
          <Button size="sm" variant="primary" onClick={() => run(false)}>
            Make clips
          </Button>
          <Button size="sm" variant="secondary" onClick={() => run(true)}>
            Remake all
          </Button>
        </div>
      </SettingRow>
    </div>
  );
};

// A button for one tag's page: cut its clip again.
export const TagClipButton: React.FC<{ tagID: string }> = ({ tagID }) => {
  const Toast = useToast();
  const [make, { loading }] = GQL.useMakeTagClipsMutation();
  return (
    <Button
      variant="secondary"
      disabled={loading}
      title="Cut this tag's looping clip again from its best scene"
      onClick={async () => {
        try {
          await make({ variables: { tag_ids: [tagID] } });
          Toast.success("Making a new clip. Reload the page in a few seconds.");
        } catch (e) {
          Toast.error(e);
        }
      }}
    >
      Make new clip
    </Button>
  );
};
