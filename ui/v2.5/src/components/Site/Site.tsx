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
import {
  CategoriesPage,
  CategoryPage,
  ChannelPage,
  ChannelsPage,
  ModelPage,
  ModelsPage,
  MyListPage,
  ScenePage,
  ScenesPage,
  SearchPage,
} from "./SitePages";
import { findSiteStyle, siteStyles } from "./siteStyles";
import "./site.scss";

const SiteHeader: React.FC<{ name: string; female: boolean }> = ({ name, female }) => {
  const history = useHistory();
  const location = useLocation();
  const { isAdmin, me } = useCurrentUser();
  const [q, setQ] = useState("");
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

  function search(e: React.FormEvent) {
    e.preventDefault();
    const text = q.trim();
    if (text) history.push(`/site/search?q=${encodeURIComponent(text)}`);
  }

  const words = name.trim().split(/\s+/);
  const first = words.length > 1 ? words.slice(0, -1).join(" ") : name;
  const last = words.length > 1 ? words[words.length - 1] : "";

  return (
    <header className={`site-header ${scrolled ? "scrolled" : ""}`}>
      <div className="site-header-inner">
        <button type="button" className="site-burger" onClick={() => setMobile(!mobile)} aria-label="Menu">
          ☰
        </button>
        <Link to="/site" className="site-logo">
          {first}
          {last && <span>{last}</span>}
        </Link>
        <nav className={`site-nav ${mobile ? "open" : ""}`}>
          <NavLink exact to="/site">
            Home
          </NavLink>
          <NavLink to="/site/scenes">Scenes</NavLink>
          <NavLink to="/site/models">{female ? "Pornstars" : "Models"}</NavLink>
          <NavLink to="/site/categories">Categories</NavLink>
          <NavLink to="/site/channels">Channels</NavLink>
          <NavLink to="/site/my-list">My list</NavLink>
        </nav>
        <form className="site-search" onSubmit={search}>
          <input
            type="search"
            placeholder="Search scenes and stars"
            value={q}
            onChange={(e) => setQ(e.currentTarget.value)}
          />
          <button type="submit" aria-label="Search">
            ⌕
          </button>
        </form>
        <div className="site-account">
          <button type="button" className="site-avatar" onClick={() => setMenu(!menu)}>
            {(me?.username ?? "Me").slice(0, 1).toUpperCase()}
          </button>
          {menu && (
            <div className="site-account-menu">
              <div className="site-account-name">{me?.username ?? "Owner"}</div>
              <Link to="/site/my-list">My list</Link>
              <Link to="/site/account">Account</Link>
              {isAdmin && <Link to="/">Admin interface</Link>}
              <a href={`${baseURL}logout`}>Log out</a>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};

const SiteFooter: React.FC<{ name: string; tagline: string }> = ({ name, tagline }) => (
  <footer className="site-footer">
    <div className="site-page">
      <div className="site-footer-brand">{name}</div>
      {tagline && <div className="site-footer-tagline">{tagline}</div>}
      <nav>
        <Link to="/site/scenes">Scenes</Link>
        <Link to="/site/models">Models</Link>
        <Link to="/site/categories">Categories</Link>
        <Link to="/site/channels">Channels</Link>
      </nav>
      <div className="site-footer-small">All models were 18 years of age or older at the time of filming.</div>
    </div>
  </footer>
);

const AccountPage: React.FC = () => (
  <div className="site-page site-account-page">
    <h1 className="site-page-title">Account</h1>
    <div className="site-account-panel">
      <AccountPanel />
    </div>
  </div>
);

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

  useEffect(() => {
    document.body.classList.add("site-mode", `site-body-${style.id}`);
    return () => document.body.classList.remove("site-mode", `site-body-${style.id}`);
  }, [style.id]);

  useEffect(() => {
    window.scrollTo({ top: 0 });
  }, [location.pathname]);

  if (loading && !config) {
    return (
      <div className={`site-root site-style-${style.id}`}>
        <Loading />
      </div>
    );
  }
  if (!config) return null;

  const female = config.models_gender === "FEMALE";

  return (
    <SiteContext.Provider value={{ config, style }}>
      <div className={`site-root site-style-${style.id}`}>
        <Helmet>
          <title>{config.name}</title>
        </Helmet>
        {previewStyle && previewStyle !== config.style && (
          <div className="site-preview-bar">
            Previewing the {style.name} style.{" "}
            <button type="button" onClick={() => setPreviewStyle(null)}>
              Stop preview
            </button>
          </div>
        )}
        <SiteHeader name={config.name} female={female} />
        <main className="site-main">
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
            <Route path="/site/account" component={AccountPage} />
            <Route component={SiteHome} />
          </Switch>
        </main>
        <SiteFooter name={config.name} tagline={config.tagline} />
      </div>
    </SiteContext.Provider>
  );
};

export default Site;
