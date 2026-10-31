import React, { useEffect, useState } from "react";
import { Button, Form } from "react-bootstrap";
import { faArrowDown, faArrowUp, faTimes } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { useToast } from "src/hooks/Toast";
import { defaultRowTitle } from "src/components/Site/SiteHome";
import { findSiteStyle } from "src/components/Site/siteStyles";
import { SettingRow } from "./util";

type Cfg = GQL.SiteConfigDataFragment;

interface IRow {
  type: string;
  title: string;
  tagID?: string;
  tagName?: string;
  enabled: boolean;
}

const rowHelp: Record<string, string> = {
  continue: "Scenes the viewer started but did not finish",
  latest: "Newest additions, as a grid",
  models: "Ranked pornstars with the most scenes",
  trending: "Most watched by everyone in the last two weeks",
  popular: "Most watched by everyone, all time",
  recommended: "Picked from what the viewer watches",
  favorites: "New scenes with the viewer's favourite pornstars",
  liked: "Scenes the viewer liked",
  reels: "Short clips; opens the Reels page",
  top_rated: "Best average rating, as big tiles",
  categories: "Most used categories",
  channels: "Studios with the most scenes",
  tag: "Newest scenes with a tag",
};

export const HomeRowsEditor: React.FC<{
  cfg: Cfg;
  tags: { id: string; name: string }[];
  save: (input: GQL.SiteConfigInput, message?: string) => Promise<void>;
}> = ({ cfg, tags, save }) => {
  const female = cfg.models_gender === "FEMALE";
  const fromCfg = (): IRow[] =>
    cfg.home_rows.map((r) => ({
      type: r.type,
      title: r.title,
      tagID: r.tag?.id,
      tagName: r.tag?.name,
      enabled: r.enabled,
    }));
  const [rows, setRows] = useState<IRow[]>(fromCfg);
  const [dirty, setDirty] = useState(false);
  const [addTag, setAddTag] = useState("");
  useEffect(() => {
    setRows(fromCfg());
    setDirty(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cfg.home_rows]);

  function change(next: IRow[]) {
    setRows(next);
    setDirty(true);
  }

  function move(i: number, d: number) {
    const j = i + d;
    if (j < 0 || j >= rows.length) return;
    const next = [...rows];
    [next[i], next[j]] = [next[j], next[i]];
    change(next);
  }

  return (
    <div className="setting-section">
      <h1>Home page rows</h1>
      <div className="sub-heading">
        Choose which rows the home page shows and in what order. Rows with
        nothing in them are hidden for that viewer. Add rows for any tag, for
        example a &quot;Featured&quot; or &quot;Editor&apos;s picks&quot; tag.
      </div>
      <div className="stasharr-site-rows">
        {rows.map((r, i) => (
          <div key={`${r.type}-${r.tagID ?? ""}-${i}`} className={`stasharr-site-row ${r.enabled ? "" : "off"}`}>
            <Form.Check
              id={`site-row-${i}`}
              checked={r.enabled}
              onChange={() => change(rows.map((x, k) => (k === i ? { ...x, enabled: !x.enabled } : x)))}
            />
            <div className="stasharr-site-row-main">
              <Form.Control
                className="text-input"
                size="sm"
                value={r.title}
                placeholder={r.type === "tag" ? r.tagName : defaultRowTitle(r.type, female)}
                onChange={(e) => {
                  const v = e.currentTarget.value;
                  change(rows.map((x, k) => (k === i ? { ...x, title: v } : x)));
                }}
              />
              <small className="text-muted">
                {r.type === "tag" ? `Tag: ${r.tagName}` : rowHelp[r.type]}
              </small>
            </div>
            <Button variant="link" size="sm" title="Move up" disabled={i === 0} onClick={() => move(i, -1)}>
              <Icon icon={faArrowUp} />
            </Button>
            <Button variant="link" size="sm" title="Move down" disabled={i === rows.length - 1} onClick={() => move(i, 1)}>
              <Icon icon={faArrowDown} />
            </Button>
            {r.type === "tag" ? (
              <Button variant="link" size="sm" title="Remove row" onClick={() => change(rows.filter((_, k) => k !== i))}>
                <Icon icon={faTimes} />
              </Button>
            ) : (
              <span className="stasharr-site-row-spacer" />
            )}
          </div>
        ))}
      </div>
      <div className="stasharr-site-row-add">
        <Form.Control as="select" className="input-control" value={addTag} onChange={(e) => setAddTag(e.currentTarget.value)}>
          <option value="">Add a row for a tag…</option>
          {tags.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </Form.Control>
        <Button
          variant="secondary"
          disabled={!addTag}
          onClick={() => {
            const t = tags.find((x) => x.id === addTag);
            if (!t) return;
            change([{ type: "tag", title: "", tagID: t.id, tagName: t.name, enabled: true }, ...rows]);
            setAddTag("");
          }}
        >
          Add
        </Button>
      </div>
      {dirty && (
        <div className="mt-3">
          <Button
            variant="primary"
            onClick={() =>
              save(
                {
                  home_rows: rows.map((r) => ({
                    type: r.type,
                    title: r.title,
                    tag_id: r.tagID,
                    enabled: r.enabled,
                  })),
                },
                "Home page saved"
              )
            }
          >
            Save home page
          </Button>{" "}
          <Button variant="secondary" onClick={() => { setRows(fromCfg()); setDirty(false); }}>
            Discard changes
          </Button>
        </div>
      )}
    </div>
  );
};

export const PromoEditor: React.FC<{
  cfg: Cfg;
  save: (input: GQL.SiteConfigInput, message?: string) => Promise<void>;
}> = ({ cfg, save }) => {
  const [p, setP] = useState({ ...cfg.promo });
  useEffect(() => setP({ ...cfg.promo }), [cfg.promo]);
  const input = { enabled: p.enabled, text: p.text, link: p.link, image: p.image };
  const dirty = JSON.stringify(input) !== JSON.stringify({ enabled: cfg.promo.enabled, text: cfg.promo.text, link: cfg.promo.link, image: cfg.promo.image });

  return (
    <div className="setting-section">
      <h1>Promo banner</h1>
      <div className="sub-heading">
        A wide banner under the ticker, for example &quot;New: the Summer
        collection&quot; linking to a category or channel.
      </div>
      <SettingRow heading="Show the banner">
        <Form.Switch id="site-promo" checked={p.enabled} onChange={() => setP({ ...p, enabled: !p.enabled })} />
      </SettingRow>
      <SettingRow heading="Text">
        <Form.Control className="text-input" value={p.text} onChange={(e) => setP({ ...p, text: e.currentTarget.value })} />
      </SettingRow>
      <SettingRow heading="Link" sub="A site page such as /site/category/12 or /site/model/34, or a full https:// address.">
        <Form.Control className="text-input" value={p.link} onChange={(e) => setP({ ...p, link: e.currentTarget.value })} />
      </SettingRow>
      <SettingRow heading="Background picture" sub="Optional. The address of a picture, for example a scene's screenshot.">
        <Form.Control className="text-input" value={p.image} onChange={(e) => setP({ ...p, image: e.currentTarget.value })} />
      </SettingRow>
      {dirty && (
        <Button variant="primary" onClick={() => save({ promo: input }, "Promo banner saved")}>
          Save promo banner
        </Button>
      )}
    </div>
  );
};

function readFile(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result));
    r.onerror = () => reject(r.error);
    r.readAsDataURL(file);
  });
}

export const BrandingEditor: React.FC<{
  cfg: Cfg;
  save: (input: GQL.SiteConfigInput, message?: string) => Promise<void>;
}> = ({ cfg, save }) => {
  const Toast = useToast();
  const [setLogo, { loading }] = GQL.useSiteSetLogoMutation({ refetchQueries: ["SiteConfig"] });
  const styleAccent = findSiteStyle(cfg.style).swatches[2];
  const [accent, setAccent] = useState(cfg.accent || styleAccent);
  useEffect(() => setAccent(cfg.accent || styleAccent), [cfg.accent, styleAccent]);

  async function upload(file: File | undefined) {
    if (!file) return;
    if (file.size > 2 * 1024 * 1024) {
      Toast.error("Keep the picture under 2 MB");
      return;
    }
    try {
      await setLogo({ variables: { data: await readFile(file) } });
      Toast.success("Logo saved");
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div className="setting-section">
      <h1>Branding</h1>
      <SettingRow heading="Logo" sub="A PNG, SVG or WebP with a transparent background works best. Shown instead of the site name.">
        <div className="stasharr-site-logo-edit">
          {cfg.logo_url && <img src={cfg.logo_url} alt="Logo" />}
          <Form.Control
            type="file"
            accept="image/png,image/jpeg,image/webp,image/svg+xml"
            disabled={loading}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => upload(e.currentTarget.files?.[0])}
          />
          {cfg.logo_url && (
            <Button variant="danger" size="sm" disabled={loading} onClick={() => setLogo({ variables: { data: "" } })}>
              Remove logo
            </Button>
          )}
        </div>
      </SettingRow>
      <SettingRow heading="Accent colour" sub="Buttons, highlights and badges. Leave it to use the style's own colour.">
        <div className="stasharr-site-accent">
          <input type="color" value={accent} onChange={(e) => setAccent(e.currentTarget.value)} />
          <Button
            size="sm"
            variant="primary"
            disabled={accent === (cfg.accent || styleAccent)}
            onClick={() => save({ accent }, "Accent colour saved")}
          >
            Use this colour
          </Button>
          {cfg.accent && (
            <Button size="sm" variant="secondary" onClick={() => save({ accent: "" }, "Back to the style's colour")}>
              Use the style&apos;s colour
            </Button>
          )}
        </div>
      </SettingRow>
    </div>
  );
};
