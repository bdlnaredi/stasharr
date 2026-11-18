import React, { useEffect, useMemo, useState } from "react";
import { Link, NavLink, Route, Switch, useHistory, useLocation } from "react-router-dom";
import { Helmet } from "react-helmet";
import * as GQL from "src/core/generated-graphql";
import { baseURL } from "src/core/createClient";
import { useCurrentUser } from "src/components/Stasharr/CurrentUser";
import { AccountPanel } from "src/components/Stasharr/AccountPanel";
import { Loading } from "./SiteCards";
import { SiteContext } from "./SiteContext";
import { SiteHome } from "./SiteHome";
import { SiteSearch } from "./SiteSearch";
import { ReelsPage } from "./SiteReels";
import {
  CategoriesPage,
  CategoryPage,
  ChannelPage,
  ChannelsPage,
  ModelPage,
  ModelsPage,
  MyListPage,
  PlaylistPage,
  ScenePage,
  ScenesPage,
  SearchPage,
} from "./SitePages";
import {
  ISiteLayout,
  findSiteLayout,
  findSiteStyle,
  setViewerLayout,
  siteLayouts,
  siteStyles,
  viewerLayout,
} from "./siteStyles";
import { SiteIcon } from "./SiteIcons";
import { Lang, getLang, languages, setLang, t } from "./i18n";
import { SiteAlerts } from "./SiteAlerts";
import { Discreet, DiscreetSettings, hideNow, loadDiscreet } from "./SiteDiscreet";
import { MiniPlayer, MiniPlayerProvider, miniPlayerOn, setMiniPlayerOn } from "./SiteMiniPlayer";
import { ProfileAvatar, ProfilesPage, profileWasPicked, useProfileSwitch } from "./SiteProfiles";
import { RecapPage } from "./SiteRecap";
import "./site.scss";
import "./siteLayouts.scss";
import "./siteExtras.scss";

const LangPicker: React.FC<{ lang: Lang; onChange: (l: Lang) => void }> = ({ lang, onChange }) => (
  <label className="site-lang" title={t("Language")}>
    <span aria-hidden="true">🌐</span>
    <select value={lang} aria-label={t("Language")} onChange={(e) => onChange(e.target.value as Lang)}>
      {languages.map((l) => (
        <option key={l.id} value={l.id}>
          {l.label}
        </option>
      ))}
    </select>
    <span className="site-lang-code">{lang.toUpperCase()}</span>
  </label>
);

// The menu, shared by the top bar, the side menu and the dock.
const NavLinks: React.FC<{ female: boolean }> = ({ female }) => {
  const { canViewReels } = useCurrentUser();
  return (
    <>
      <NavLink exact to="/site">
        <SiteIcon name="home" />
        <span>{t("Home")}</span>
      </NavLink>
      <NavLink to="/site/scenes">
        <SiteIcon name="scenes" />
        <span>{t("Scenes")}</span>
      </NavLink>
      <NavLink to="/site/models">
        <SiteIcon name="models" />
        <span>{t(female ? "Pornstars" : "Models")}</span>
      </NavLink>
      <NavLink to="/site/categories">
        <SiteIcon name="categories" />
        <span>{t("Categories")}</span>
      </NavLink>
      <NavLink to="/site/channels">
        <SiteIcon name="channels" />
        <span>{t("Channels")}</span>
      </NavLink>
      {canViewReels && (
        <NavLink to="/site/reels">
          <SiteIcon name="reels" />
          <span>{t("Reels")}</span>
        </NavLink>
      )}
      <NavLink to="/site/my-list">
        <SiteIcon name="list" />
        <span>{t("My list")}</span>
      </NavLink>
    </>
  );
};

const Logo: React.FC<{ name: string; logo?: string | null }> = ({ name, logo }) => {
  const words = name.trim().split(/\s+/);
  const first = words.length > 1 ? words.slice(0, -1).join(" ") : name;
  const last = words.length > 1 ? words[words.length - 1] : "";
  return (
    <Link to="/site" className={`site-logo ${logo ? "site-logo-image" : ""}`}>
      {logo ? (
        <img src={logo} alt={name} />
      ) : (
        <>
          {first}
          {last && <span>{last}</span>}
        </>
      )}
    </Link>
  );
};

// Immersive: the menu as a floating dock at the bottom
const Dock: React.FC<{ female: boolean }> = ({ female }) => (
  <nav className="site-dock">
    <NavLinks female={female} />
  </nav>
);

const LayoutMenu: React.FC<{ layout: ISiteLayout; own: boolean; onPick: (id: string | null) => void }> = ({
  layout,
  own,
  onPick,
}) => (
  <div className="site-layout-menu">
    <small>{t("Layout")}</small>
    <button type="button" className={!own ? "on" : ""} onClick={() => onPick(null)}>
      {t("Site default")}
    </button>
    {siteLayouts.map((l) => (
      <button type="button" key={l.id} className={own && l.id === layout.id ? "on" : ""} onClick={() => onPick(l.id)}>
        {t(l.name)}
      </button>
    ))}
  </div>
);

const SiteHeader: React.FC<{
  name: string;
  female: boolean;
  logo?: string | null;
  lang: Lang;
  onLang: (l: Lang) => void;
  layout: ISiteLayout;
  ownLayout: boolean;
  onLayout: (id: string | null) => void;
}> = ({ name, female, logo, lang, onLang, layout, ownLayout, onLayout }) => {
  const location = useLocation();
  const { isAdmin, me } = useCurrentUser();
  const [menu, setMenu] = useState(false);
  const [mobile, setMobile] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 40);
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  // close menus when the page changes
  useEffect(() => {
    setMenu(false);
    setMobile(false);
  }, [location.pathname]);

  const topMenu = layout.nav === "top";
  const profiles = GQL.useSiteProfilesQuery({ skip: !me?.multi_user_enabled, fetchPolicy: "cache-and-network" });
  const family = profiles.data?.siteProfiles ?? [];
  const current = family.find((p) => p.is_current);
  const others = family.filter((p) => !p.is_current);
  const { go, dialog } = useProfileSwitch();
  const discreetOn = loadDiscreet().on;

  return (
    <header className={`site-header ${scrolled ? "scrolled" : ""}`}>
      <div className="site-header-inner">
        {topMenu && (
          <button type="button" className="site-burger" onClick={() => setMobile(!mobile)} aria-label={t("Menu")}>
            ☰
          </button>
        )}
        <Logo name={name} logo={logo} />
        {topMenu && (
          <nav className={`site-nav ${mobile ? "open" : ""}`}>
            <NavLinks female={female} />
          </nav>
        )}
        <div className="site-header-tools">
          <SiteSearch />
          <LangPicker lang={lang} onChange={onLang} />
          <SiteAlerts />
          <div className="site-account">
            <button
              type="button"
              className="site-avatar"
              style={current ? { background: profileBg(current) } : undefined}
              onClick={() => setMenu(!menu)}
              aria-label={t("Account")}
            >
              {(me?.display_name || me?.username || "Me").slice(0, 1).toUpperCase()}
            </button>
            {menu && (
              <div className="site-account-menu">
                <div className="site-account-name">{me?.display_name || me?.username || "Owner"}</div>
                {others.length > 0 && (
                  <div className="site-account-profiles">
                    {others.map((p) => (
                      <button type="button" key={p.id} onClick={() => go(p)}>
                        <ProfileAvatar profile={p} size={26} />
                        <span>{p.name}</span>
                        {p.has_pin && <small>🔒</small>}
                      </button>
                    ))}
                  </div>
                )}
                {me?.multi_user_enabled && <Link to="/site/profiles">{t("Manage profiles")}</Link>}
                <Link to="/site/my-list">{t("My list")}</Link>
                <Link to="/site/recap">{t("Monthly recap")}</Link>
                <Link to="/site/account">{t("Account")}</Link>
                {discreetOn && (
                  <button type="button" className="site-account-hide" onClick={() => hideNow()}>
                    {t("Hide now")}
                  </button>
                )}
                {isAdmin && <Link to="/">{t("Admin interface")}</Link>}
                <a href={`${baseURL}logout`}>{t("Log out")}</a>
                <LayoutMenu
                  layout={layout}
                  own={ownLayout}
                  onPick={(id) => {
                    onLayout(id);
                    setMenu(false);
                  }}
                />
              </div>
            )}
          </div>
        </div>
      </div>
      {dialog}
    </header>
  );
};

function profileBg(p: { id: string; color: string }) {
  return p.color || undefined;
}

const SiteFooter: React.FC<{ name: string; tagline: string }> = ({ name, tagline }) => (
  <footer className="site-footer">
    <div className="site-page">
      <div className="site-footer-brand">{name}</div>
      {tagline && <div className="site-footer-tagline">{tagline}</div>}
      <nav>
        <Link to="/site/scenes">{t("Scenes")}</Link>
        <Link to="/site/models">{t("Models")}</Link>
        <Link to="/site/categories">{t("Categories")}</Link>
        <Link to="/site/channels">{t("Channels")}</Link>
      </nav>
      <div className="site-footer-small">{t("All models were 18 years of age or older at the time of filming.")}</div>
    </div>
  </footer>
);

const AccountPage: React.FC = () => {
  const { me } = useCurrentUser();
  const [mini, setMini] = useState(miniPlayerOn);
  return (
    <div className="site-page site-account-page">
      <h1 className="site-page-title">{t("Account")}</h1>
      {me?.is_profile ? (
        <p className="site-muted">
          {t("This is a profile. Switch to the main profile to change the password or sign-in settings.")}{" "}
          <Link to="/site/profiles">{t("Profiles")}</Link>
        </p>
      ) : (
        <div className="site-account-panel">
          <AccountPanel />
        </div>
      )}
      <section className="site-discreet-settings">
        <h2>{t("Mini-player")}</h2>
        <label className="site-switch">
          <input
            type="checkbox"
            checked={mini}
            onChange={(e) => {
              setMiniPlayerOn(e.target.checked);
              setMini(e.target.checked);
            }}
          />
          <span>{t("Keep playing in a small window when I leave a scene")}</span>
        </label>
      </section>
      <DiscreetSettings />
    </div>
  );
};

const AGE_KEY = "stasharr-site-adult-ok";

function ageConfirmed(): boolean {
  try {
    return localStorage.getItem(AGE_KEY) === "1";
  } catch {
    return false;
  }
}

const AgeGate: React.FC<{ name: string; logo?: string | null; onEnter: () => void }> = ({ name, logo, onEnter }) => (
  <div className="site-age-gate">
    <div className="site-age-card">
      {logo ? <img src={logo} alt={name} className="site-age-logo" /> : <div className="site-logo">{name}</div>}
      <h2>{t("This is an adult website")}</h2>
      <p>
        {t(
          "It contains sexually explicit material. Only continue if you are at least 18 years old, or the age of majority where you live, and you want to see it."
        )}
      </p>
      <div className="site-age-actions">
        <button
          type="button"
          className="site-btn site-btn-primary"
          onClick={() => {
            try {
              localStorage.setItem(AGE_KEY, "1");
            } catch {
              // private window: asked again next time
            }
            onEnter();
          }}
        >
          {t("I am 18 or older — Enter")}
        </button>
        <a className="site-btn site-btn-ghost" href="https://www.google.com" rel="noreferrer">
          {t("Leave")}
        </a>
      </div>
    </div>
  </div>
);

// Readable text colour on top of an accent colour.
function textOn(hex: string): string {
  const n = Number.parseInt(hex.slice(1), 16);
  const r = (n >> 16) & 255;
  const g = (n >> 8) & 255;
  const b = n & 255;
  return 0.299 * r + 0.587 * g + 0.114 * b > 160 ? "#111111" : "#ffffff";
}

// Asks "Who's watching?" once per browser session when there are profiles.
const WhoIsWatching: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
  const { me } = useCurrentUser();
  const { data } = GQL.useSiteProfilesQuery({ skip: !me?.multi_user_enabled });
  const count = data?.siteProfiles.length ?? 0;
  useEffect(() => {
    if (count > 1 && !profileWasPicked() && location.pathname !== "/site/profiles") {
      history.replace("/site/profiles");
    }
  }, [count, history, location.pathname]);
  return null;
};

const Site: React.FC = () => {
  const location = useLocation();
  const { data, loading } = GQL.useSiteConfigQuery({ fetchPolicy: "cache-and-network" });
  const config = data?.siteConfig;

  // admins can preview another style with ?style=neon
  const preview = useMemo(() => {
    const s = new URLSearchParams(location.search).get("style");
    return s && siteStyles.some((x) => x.id === s) ? s : null;
  }, [location.search]);
  const [previewStyle, setPreviewStyle] = useState<string | null>(null);
  useEffect(() => {
    if (preview) setPreviewStyle(preview);
  }, [preview]);

  const style = findSiteStyle(previewStyle ?? config?.style);

  // admins can preview a layout with ?layout=bento; each viewer can also
  // pick their own from the account menu
  const previewLayoutParam = useMemo(() => {
    const l = new URLSearchParams(location.search).get("layout");
    return l && siteLayouts.some((x) => x.id === l) ? l : null;
  }, [location.search]);
  const [previewLayout, setPreviewLayout] = useState<string | null>(null);
  useEffect(() => {
    if (previewLayoutParam) setPreviewLayout(previewLayoutParam);
  }, [previewLayoutParam]);
  const [ownLayout, setOwnLayout] = useState<string | null>(viewerLayout);
  const pickLayout = (id: string | null) => {
    setViewerLayout(id);
    setOwnLayout(id);
    setPreviewLayout(null);
  };
  const layout = findSiteLayout(previewLayout ?? ownLayout ?? config?.layout);
  const [adult, setAdult] = useState(ageConfirmed);
  const [lang, setLangState] = useState<Lang>(getLang);
  const changeLang = (l: Lang) => {
    setLang(l);
    setLangState(l);
  };

  useEffect(() => {
    document.body.classList.add("site-mode", `site-body-${style.id}`, `site-body-layout-${layout.id}`);
    return () =>
      document.body.classList.remove("site-mode", `site-body-${style.id}`, `site-body-layout-${layout.id}`);
  }, [style.id, layout.id]);

  useEffect(() => {
    window.scrollTo({ top: 0 });
  }, [location.pathname]);

  if (loading && !config) {
    return (
      <div className={`site-root site-style-${style.id} site-layout-${layout.id}`}>
        <Loading />
      </div>
    );
  }
  if (!config) return null;

  const female = config.models_gender === "FEMALE";
  const accentVars = config.accent
    ? ({
        "--site-accent": config.accent,
        "--site-accent-2": config.accent,
        "--site-on-accent": textOn(config.accent),
      } as React.CSSProperties)
    : undefined;

  return (
    <SiteContext.Provider value={{ config, style, layout }}>
      <MiniPlayerProvider>
      <div className={`site-root site-style-${style.id} site-layout-${layout.id}`} style={accentVars}>
        <WhoIsWatching />
        {config.age_gate && !adult && (
          <AgeGate name={config.name} logo={config.logo_url} onEnter={() => setAdult(true)} />
        )}
        <Helmet>
          <html lang={lang} />
          <title>{config.name}</title>
        </Helmet>
        {previewStyle && previewStyle !== config.style && (
          <div className="site-preview-bar">
            {t("Previewing the {name} style.", { name: style.name })}{" "}
            <button type="button" onClick={() => setPreviewStyle(null)}>
              {t("Stop preview")}
            </button>
          </div>
        )}
        {previewLayout && previewLayout !== config.layout && (
          <div className="site-preview-bar">
            {t("Previewing the {name} layout.", { name: t(layout.name) })}{" "}
            <button type="button" onClick={() => setPreviewLayout(null)}>
              {t("Stop preview")}
            </button>
          </div>
        )}
        <SiteHeader
          name={config.name}
          female={female}
          logo={config.logo_url}
          lang={lang}
          onLang={changeLang}
          layout={layout}
          ownLayout={!!ownLayout}
          onLayout={pickLayout}
        />
        {/* keyed on the language so every page redraws in the new one */}
        <main className="site-main" key={lang}>
          <Switch>
            <Route exact path="/site" component={SiteHome} />
            <Route path="/site/scenes" component={ScenesPage} />
            <Route path="/site/scene/:id" component={ScenePage} />
            <Route path="/site/models" component={ModelsPage} />
            <Route path="/site/model/:id" component={ModelPage} />
            <Route path="/site/channels" component={ChannelsPage} />
            <Route path="/site/studio/:id" component={ChannelPage} />
            <Route path="/site/categories" component={CategoriesPage} />
            <Route path="/site/category/:id" component={CategoryPage} />
            <Route path="/site/search" component={SearchPage} />
            <Route path="/site/my-list" component={MyListPage} />
            <Route path="/site/reels" component={ReelsPage} />
            <Route path="/site/account" component={AccountPage} />
            <Route path="/site/profiles" component={ProfilesPage} />
            <Route path="/site/recap" component={RecapPage} />
            <Route path="/site/playlist/:id" component={PlaylistPage} />
            <Route component={SiteHome} />
          </Switch>
        </main>
        <SiteFooter key={`f-${lang}`} name={config.name} tagline={config.tagline} />
        {layout.nav === "dock" && <Dock female={female} />}
        <MiniPlayer />
        <Discreet />
      </div>
      </MiniPlayerProvider>
    </SiteContext.Provider>
  );
};

export default Site;
