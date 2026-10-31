import React, { useEffect, useState } from "react";
import { Button, Form } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { baseURL } from "src/core/createClient";
import { siteStyles } from "src/components/Site/siteStyles";
import { SettingRow } from "./util";

const featuredOptions = [
  { id: "top_rated", label: "Top rated by everyone" },
  { id: "popular", label: "Most watched" },
  { id: "recent", label: "Newest additions" },
  { id: "random", label: "Random each visit" },
  { id: "tag", label: "Scenes with a tag I choose" },
];

export const SitePanel: React.FC = () => {
  const Toast = useToast();
  const { data, loading } = GQL.useSiteConfigQuery({ fetchPolicy: "network-only" });
  const [configure, { loading: saving }] = GQL.useConfigureSiteMutation({
    refetchQueries: ["SiteConfig"],
  });
  const tags = GQL.useSiteTagsQuery({
    variables: {
      filter: { per_page: 500, sort: "name", direction: GQL.SortDirectionEnum.Asc },
      tag_filter: { scene_count: { value: 0, modifier: GQL.CriterionModifier.GreaterThan } },
    },
  });
  const cfg = data?.siteConfig;
  const [name, setName] = useState("");
  const [tagline, setTagline] = useState("");

  useEffect(() => {
    if (cfg) {
      setName(cfg.name);
      setTagline(cfg.tagline);
    }
  }, [cfg]);

  if (loading && !cfg) return <LoadingIndicator />;
  if (!cfg) return null;

  async function save(input: GQL.SiteConfigInput, message?: string) {
    try {
      await configure({ variables: { input } });
      if (message) Toast.success(message);
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div className="stasharr-site-panel">
      <div className="setting-section">
        <div className="stasharr-section-head">
          <h1>Site look</h1>
          <Button variant="secondary" href={`${baseURL}site`} target="_blank" rel="noopener">
            Open the site
          </Button>
        </div>
        <div className="sub-heading">
          Turns the library into a storefront: a big featured banner, a
          &quot;just added&quot; ticker, top pornstars, trending and top rated
          scenes, categories and channels. Users it applies to see only the
          site when they log in; the normal interface is hidden from them.
          Admins always keep the normal interface and can open the site from
          the menu.
        </div>

        <SettingRow
          heading="Use the site"
          sub={cfg.enabled ? "On." : "Off. Everyone sees the normal interface."}
        >
          <Form.Switch
            id="site-enabled"
            disabled={saving}
            checked={cfg.enabled}
            onChange={() => save({ enabled: !cfg.enabled })}
          />
        </SettingRow>

        <SettingRow heading="Show it to">
          <Form.Control
            as="select"
            className="input-control"
            disabled={saving}
            value={cfg.audience}
            onChange={(e) => save({ audience: e.currentTarget.value })}
          >
            <option value="viewers">Viewers</option>
            <option value="non_admins">Viewers and editors</option>
          </Form.Control>
        </SettingRow>

        <SettingRow heading="Site name" sub="Shown as the logo. The last word is coloured.">
          <Form.Control
            className="text-input"
            value={name}
            disabled={saving}
            onChange={(e) => setName(e.currentTarget.value)}
            onBlur={() => name !== cfg.name && save({ name })}
          />
        </SettingRow>

        <SettingRow heading="Tagline" sub="Shown in the footer.">
          <Form.Control
            className="text-input"
            value={tagline}
            disabled={saving}
            onChange={(e) => setTagline(e.currentTarget.value)}
            onBlur={() => tagline !== cfg.tagline && save({ tagline })}
          />
        </SettingRow>

        <SettingRow heading="Featured banner" sub="Which scenes appear in the big banner at the top.">
          <Form.Control
            as="select"
            className="input-control"
            disabled={saving}
            value={cfg.featured}
            onChange={(e) => save({ featured: e.currentTarget.value })}
          >
            {featuredOptions.map((o) => (
              <option key={o.id} value={o.id}>
                {o.label}
              </option>
            ))}
          </Form.Control>
        </SettingRow>

        {cfg.featured === "tag" && (
          <SettingRow
            heading="Featured tag"
            sub='Tag the scenes you want to feature, for example with a "Featured" tag.'
          >
            <Form.Control
              as="select"
              className="input-control"
              disabled={saving}
              value={cfg.featured_tag?.id ?? ""}
              onChange={(e) => save({ featured_tag_id: e.currentTarget.value })}
            >
              <option value="">Choose a tag</option>
              {(tags.data?.findTags.tags ?? []).map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </Form.Control>
          </SettingRow>
        )}

        <SettingRow heading="Models pages show">
          <Form.Control
            as="select"
            className="input-control"
            disabled={saving}
            value={cfg.models_gender}
            onChange={(e) => save({ models_gender: e.currentTarget.value })}
          >
            <option value="FEMALE">Female pornstars only</option>
            <option value="ALL">Everyone</option>
          </Form.Control>
        </SettingRow>
      </div>

      <div className="setting-section">
        <h1>Style</h1>
        <div className="stasharr-site-styles">
          {siteStyles.map((s) => (
            <div
              key={s.id}
              className={`stasharr-site-style ${s.id === cfg.style ? "selected" : ""}`}
            >
              <div className="stasharr-site-swatch" style={{ background: s.swatches[0] }}>
                <div className="stasharr-site-swatch-bar" style={{ background: s.swatches[1] }}>
                  <span style={{ color: s.swatches[3] }}>{s.name}</span>
                  <i style={{ background: s.swatches[2] }} />
                </div>
                <div className="stasharr-site-swatch-hero" style={{ borderColor: s.swatches[2] }}>
                  <b style={{ background: s.swatches[2] }} />
                  <em style={{ background: s.swatches[3] }} />
                </div>
                <div className="stasharr-site-swatch-cards">
                  {[0, 1, 2, 3].map((i) => (
                    <span key={i} style={{ background: s.swatches[1], borderColor: s.swatches[2] }} />
                  ))}
                </div>
              </div>
              <div className="stasharr-site-style-body">
                <h5>
                  {s.name}
                  {s.id === cfg.style && <small> · in use</small>}
                </h5>
                <p>{s.description}</p>
                <Button
                  size="sm"
                  variant="secondary"
                  href={`${baseURL}site?style=${s.id}`}
                  target="_blank"
                  rel="noopener"
                >
                  Preview
                </Button>{" "}
                <Button
                  size="sm"
                  variant="primary"
                  disabled={saving || s.id === cfg.style}
                  onClick={() => save({ style: s.id }, `The site now uses ${s.name}`)}
                >
                  Use this style
                </Button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
