import React, { useState } from "react";
import { Button, Form } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { useCurrentUser } from "./CurrentUser";

type Config = GQL.LibraryWatcherConfigDataFragment;

const Row: React.FC<{ heading: string; sub?: React.ReactNode }> = ({
  heading,
  sub,
  children,
}) => (
  <div className="setting">
    <div>
      <h3>{heading}</h3>
      {sub && <div className="sub-heading">{sub}</div>}
    </div>
    <div>{children}</div>
  </div>
);

export const AutoImportPanel: React.FC = () => {
  const Toast = useToast();
  const { isAdmin } = useCurrentUser();
  const { data, loading, refetch } = GQL.useLibraryWatcherConfigQuery({
    fetchPolicy: "network-only",
  });
  const [configure] = GQL.useConfigureLibraryWatcherMutation();
  const [generateReels] = GQL.useGenerateReelsMutation();
  const [checkNow] = GQL.useLibraryWatcherCheckNowMutation();
  const [reidentify] = GQL.useLibraryWatcherReidentifyMutation();
  const [saving, setSaving] = useState(false);

  if (loading || !data) return <LoadingIndicator />;
  const cfg: Config = data.libraryWatcherConfig;

  async function save(input: GQL.LibraryWatcherConfigInput) {
    setSaving(true);
    try {
      await configure({ variables: { input } });
      await refetch();
    } catch (e) {
      Toast.error(e);
    } finally {
      setSaving(false);
    }
  }

  async function onCheckNow() {
    try {
      await checkNow();
      Toast.success(
        "Checking your library for new files. See Settings → Tasks for progress."
      );
      setTimeout(() => refetch(), 5000);
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onReidentify() {
    try {
      await reidentify();
      Toast.success(
        "Identifying scenes that have no studio or performers. See Settings → Tasks for progress."
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  async function makeAllReels(overwrite: boolean) {
    if (
      overwrite &&
      !window.confirm(
        "Delete every reel and make new ones for every scene? This can take a long time."
      )
    ) {
      return;
    }
    try {
      await generateReels({ variables: { input: overwrite ? { overwrite: true } : {} } });
      Toast.success(
        overwrite
          ? "Remaking reels for every scene. See Settings → Tasks for progress."
          : "Making reels for every scene without one. See Settings → Tasks for progress."
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  const disabled = !isAdmin || saving;

  return (
    <div className="stasharr-auto-import">
      <div className="setting-section">
        <h1>Auto-import</h1>
        <div className="sub-heading">
          New videos added to your library folders are scanned, identified and
          prepared automatically, so you never have to press Scan.
        </div>

        <Row
          heading="Watch library folders"
          sub={
            cfg.running
              ? `Watching: ${cfg.watched_paths.join(", ")}`
              : "Off. New files are only added when you run a scan."
          }
        >
          <Form.Switch
            id="watcher-enabled"
            disabled={disabled}
            checked={cfg.enabled}
            onChange={() => save({ enabled: !cfg.enabled })}
          />
        </Row>

        {cfg.running && (
          <Row
            heading="Last check"
            sub={
              cfg.last_check_at
                ? `${new Date(cfg.last_check_at).toLocaleString()}: ${cfg.last_result ?? ""}`
                : "Not yet. New files are picked up as they appear, and the whole library is checked on the timer below."
            }
          >
            <Button
              variant="secondary"
              disabled={saving}
              onClick={() => onCheckNow()}
            >
              Check now
            </Button>
          </Row>
        )}

        <Row
          heading="Wait before scanning (seconds)"
          sub="Gives downloads time to finish. The timer restarts on every change."
        >
          <Form.Control
            className="text-input"
            type="number"
            min={5}
            max={3600}
            disabled={disabled}
            defaultValue={cfg.debounce_seconds}
            onBlur={(e: React.FocusEvent<HTMLInputElement>) => {
              const v = parseInt(e.currentTarget.value, 10);
              if (!Number.isNaN(v) && v !== cfg.debounce_seconds) {
                save({ debounce_seconds: v });
              }
            }}
          />
        </Row>

        <Row
          heading="Also check every (minutes)"
          sub="Network shares (SMB/NFS) do not report new files, so the library is also checked on a timer. 0 turns this off."
        >
          <Form.Control
            className="text-input"
            type="number"
            min={0}
            max={1440}
            disabled={disabled}
            defaultValue={cfg.poll_minutes}
            onBlur={(e: React.FocusEvent<HTMLInputElement>) => {
              const v = parseInt(e.currentTarget.value, 10);
              if (!Number.isNaN(v) && v !== cfg.poll_minutes) {
                save({ poll_minutes: v });
              }
            }}
          />
        </Row>

        <Row
          heading="Identify new scenes"
          sub={
            cfg.identify_sources.length > 0 ? (
              <>
                Sources, in order:
                <ol className="mb-0">
                  {cfg.identify_sources.map((s) => (
                    <li key={s}>{s}</li>
                  ))}
                </ol>
                Change the order under Settings → Tasks → Identify, then save
                it as the default.
              </>
            ) : (
              "No sources yet. Add ThePornDB and StashDB under Settings → Metadata Providers → Stash-box endpoints."
            )
          }
        >
          <Form.Switch
            id="watcher-identify"
            disabled={disabled}
            checked={cfg.identify}
            onChange={() => save({ identify: !cfg.identify })}
          />
        </Row>

        <Row
          heading="Add missing studios, performers and tags"
          sub="When a match has a studio, performer or tag that is not in your library yet, create it. Off means only ones you already have are linked."
        >
          <Form.Switch
            id="watcher-create-missing"
            disabled={disabled}
            checked={cfg.create_missing}
            onChange={() => save({ create_missing: !cfg.create_missing })}
          />
        </Row>

        <Row
          heading="Fix incomplete scenes"
          sub="Identify again every scene that has no studio or no performers, for example ones imported before missing studios and performers were added."
        >
          <Button
            variant="secondary"
            disabled={saving}
            onClick={() => onReidentify()}
          >
            Identify again
          </Button>
        </Row>

        <Row
          heading="Generate covers, previews and sprites"
          sub="Uses your default Generate settings."
        >
          <Form.Switch
            id="watcher-generate"
            disabled={disabled}
            checked={cfg.generate_media}
            onChange={() => save({ generate_media: !cfg.generate_media })}
          />
        </Row>

        <Row heading="Make a reel for new scenes">
          <Form.Switch
            id="watcher-reels"
            disabled={disabled}
            checked={cfg.generate_reels}
            onChange={() => save({ generate_reels: !cfg.generate_reels })}
          />
        </Row>
      </div>

      <div className="setting-section">
        <h1>Audio files</h1>
        <div className="sub-heading">
          Audio files in your library folders are added as scenes, so they can
          be identified, tagged, rated and played like videos. Their cover is
          the album art, or a picture of the sound wave. They are listed under
          Audio in the menu.
        </div>
        <Row heading="Add audio files">
          <Form.Switch
            id="audio-enabled"
            disabled={disabled}
            checked={cfg.audio_enabled}
            onChange={() => save({ audio_enabled: !cfg.audio_enabled })}
          />
        </Row>
        <Row
          heading="Audio file extensions"
          sub="Separated by commas. Run a scan after changing this."
        >
          <Form.Control
            className="text-input"
            disabled={disabled || !cfg.audio_enabled}
            defaultValue={cfg.audio_extensions.join(", ")}
            key={cfg.audio_extensions.join(",")}
            onBlur={(e: React.FocusEvent<HTMLInputElement>) => {
              const list = e.currentTarget.value
                .split(",")
                .map((x) => x.trim())
                .filter(Boolean);
              if (list.join(",") !== cfg.audio_extensions.join(",")) {
                save({ audio_extensions: list });
              }
            }}
          />
        </Row>
      </div>

      <div className="setting-section">
        <h1>Reels</h1>
        <div className="sub-heading">
          Short vertical clips, each one continuous part of a scene. Every
          scene gets a few, each from a different high point: the loudest and
          busiest parts, with the parts right after markers preferred.
        </div>
        <Row heading="Reel length (seconds)" sub="Between 5 and 90.">
          <Form.Control
            className="text-input"
            type="number"
            min={5}
            max={90}
            disabled={disabled}
            defaultValue={cfg.reel_duration}
            onBlur={(e: React.FocusEvent<HTMLInputElement>) => {
              const v = parseFloat(e.currentTarget.value);
              if (!Number.isNaN(v) && v !== cfg.reel_duration) {
                save({ reel_duration: v });
              }
            }}
          />
        </Row>
        <Row
          heading="Reels per scene"
          sub="Between 1 and 5. Short scenes get fewer, so the reels do not overlap."
        >
          <Form.Control
            className="text-input"
            type="number"
            min={1}
            max={5}
            disabled={disabled}
            defaultValue={cfg.reel_count}
            onBlur={(e: React.FocusEvent<HTMLInputElement>) => {
              const v = parseInt(e.currentTarget.value, 10);
              if (!Number.isNaN(v) && v !== cfg.reel_count) {
                save({ reel_count: v });
              }
            }}
          />
        </Row>
        <Row
          heading="Make reels for the whole library"
          sub="Runs as a task. Make reels does scenes that have none yet; Remake all replaces every existing reel with the new style. This can take a while."
        >
          <Button variant="secondary" onClick={() => makeAllReels(false)}>
            Make reels
          </Button>{" "}
          <Button variant="secondary" onClick={() => makeAllReels(true)}>
            Remake all
          </Button>
        </Row>
      </div>
    </div>
  );
};

export default AutoImportPanel;
