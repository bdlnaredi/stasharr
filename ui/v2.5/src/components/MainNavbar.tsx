import React, {
  useEffect,
  useRef,
  useState,
  useCallback,
  useMemo,
} from "react";
import {
  defineMessages,
  FormattedMessage,
  MessageDescriptor,
  useIntl,
} from "react-intl";
import { Nav, Navbar, Button } from "react-bootstrap";
import { IconDefinition } from "@fortawesome/fontawesome-svg-core";
import { LinkContainer } from "react-router-bootstrap";
import { Link, NavLink, useLocation, useHistory } from "react-router-dom";
import Mousetrap from "mousetrap";

import SessionUtils from "src/utils/session";
import { Icon } from "src/components/Shared/Icon";
import { useConfigurationContext } from "src/hooks/Config";
import { ManualStateContext } from "./Help/context";
import { SettingsButton } from "./SettingsButton";
import {
  faBars,
  faChartColumn,
  faFilm,
  faHeart,
  faImage,
  faImages,
  faMapMarkerAlt,
  faMobileAlt,
  faPlayCircle,
  faQuestionCircle,
  faSignOutAlt,
  faTag,
  faTimes,
  faUser,
  faUserCircle,
  faStore,
  faVideo,
  faListUl,
  faMusic,
} from "@fortawesome/free-solid-svg-icons";
import { baseURL } from "src/core/createClient";
import * as GQL from "src/core/generated-graphql";
import { PatchComponent } from "src/patch";
import { useCurrentUser } from "./Stasharr/CurrentUser";
import { audioScenesLink } from "./Stasharr/audio";

interface IMenuItem {
  name: string;
  message: MessageDescriptor;
  href: string;
  icon: IconDefinition;
  hotkey: string;
  userCreatable?: boolean;
}
const messages = defineMessages({
  scenes: {
    id: "scenes",
    defaultMessage: "Scenes",
  },
  images: {
    id: "images",
    defaultMessage: "Images",
  },
  groups: {
    id: "groups",
    defaultMessage: "Groups",
  },
  markers: {
    id: "markers",
    defaultMessage: "Markers",
  },
  performers: {
    id: "performers",
    defaultMessage: "Performers",
  },
  studios: {
    id: "studios",
    defaultMessage: "Studios",
  },
  tags: {
    id: "tags",
    defaultMessage: "Tags",
  },
  galleries: {
    id: "galleries",
    defaultMessage: "Galleries",
  },
  sceneTagger: {
    id: "sceneTagger",
    defaultMessage: "Scene Tagger",
  },
  donate: {
    id: "donate",
    defaultMessage: "Donate",
  },
  statistics: {
    id: "statistics",
    defaultMessage: "Statistics",
  },
  reels: {
    id: "stasharr.reels",
    defaultMessage: "Reels",
  },
  playlists: {
    id: "stasharr.playlists",
    defaultMessage: "Playlists",
  },
  audio: {
    id: "stasharr.audio",
    defaultMessage: "Audio",
  },
});

const reelsMenuItem: IMenuItem = {
  name: "reels",
  message: messages.reels,
  href: "/reels",
  icon: faMobileAlt,
  hotkey: "g r",
};

const playlistsMenuItem: IMenuItem = {
  name: "playlists",
  message: messages.playlists,
  href: "/playlists",
  icon: faListUl,
  hotkey: "g w",
};

// built on first use, as it needs the scene filter options
let audioMenuItem: IMenuItem | undefined;
function getAudioMenuItem(): IMenuItem {
  if (!audioMenuItem) {
    audioMenuItem = {
      name: "audio",
      message: messages.audio,
      href: audioScenesLink(),
      icon: faMusic,
      hotkey: "g a",
    };
  }
  return audioMenuItem;
}

const allMenuItems: IMenuItem[] = [
  {
    name: "scenes",
    message: messages.scenes,
    href: "/scenes",
    icon: faPlayCircle,
    hotkey: "g s",
    userCreatable: true,
  },
  {
    name: "images",
    message: messages.images,
    href: "/images",
    icon: faImage,
    hotkey: "g i",
  },
  {
    name: "groups",
    message: messages.groups,
    href: "/groups",
    icon: faFilm,
    hotkey: "g v",
    userCreatable: true,
  },
  {
    name: "markers",
    message: messages.markers,
    href: "/scenes/markers",
    icon: faMapMarkerAlt,
    hotkey: "g k",
  },
  {
    name: "galleries",
    message: messages.galleries,
    href: "/galleries",
    icon: faImages,
    hotkey: "g l",
    userCreatable: true,
  },
  {
    name: "performers",
    message: messages.performers,
    href: "/performers",
    icon: faUser,
    hotkey: "g p",
    userCreatable: true,
  },
  {
    name: "studios",
    message: messages.studios,
    href: "/studios",
    icon: faVideo,
    hotkey: "g u",
    userCreatable: true,
  },
  {
    name: "tags",
    message: messages.tags,
    href: "/tags",
    icon: faTag,
    hotkey: "g t",
    userCreatable: true,
  },
];

const newPathsList = allMenuItems
  .filter((item) => item.userCreatable)
  .map((item) => item.href);

const MainNavbarMenuItems = PatchComponent(
  "MainNavBar.MenuItems",
  (props: React.PropsWithChildren<unknown>) => {
    return <Nav>{props.children}</Nav>;
  }
);

const MainNavbarUtilityItems = PatchComponent(
  "MainNavBar.UtilityItems",
  (props: React.PropsWithChildren<unknown>) => {
    return <>{props.children}</>;
  }
);

export const MainNavbar: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
  const { configuration } = useConfigurationContext();
  const { openManual } = React.useContext(ManualStateContext);
  const { canViewReels, isEditor, isAdmin, me } = useCurrentUser();
  const siteConfig = GQL.useSiteConfigQuery({ fetchPolicy: "cache-first", skip: !isAdmin });
  const siteOn = !!siteConfig.data?.siteConfig.enabled;

  const [expanded, setExpanded] = useState(false);

  // Show all menu items by default, unless config says otherwise
  const menuItems = useMemo(() => {
    let cfgMenuItems = configuration?.interface.menuItems;
    const withReels = (items: IMenuItem[]) => [
      ...items,
      getAudioMenuItem(),
      playlistsMenuItem,
      ...(canViewReels ? [reelsMenuItem] : []),
    ];
    if (!cfgMenuItems) {
      return withReels(allMenuItems);
    }

    // translate old movies menu item to groups
    cfgMenuItems = cfgMenuItems.map((item) => {
      if (item === "movies") {
        return "groups";
      }
      return item;
    });

    return withReels(
      allMenuItems.filter((menuItem) => cfgMenuItems!.includes(menuItem.name))
    );
  }, [configuration, canViewReels]);

  // react-bootstrap typing bug
  const navbarRef = useRef<HTMLElement | null>(null);
  const intl = useIntl();

  const maybeCollapse = useCallback((event: Event) => {
    if (
      navbarRef.current &&
      event.target instanceof Node &&
      !navbarRef.current.contains(event.target)
    ) {
      setExpanded(false);
    }
  }, []);

  useEffect(() => {
    if (expanded) {
      document.addEventListener("click", maybeCollapse);
      document.addEventListener("touchstart", maybeCollapse);
    }
    return () => {
      document.removeEventListener("click", maybeCollapse);
      document.removeEventListener("touchstart", maybeCollapse);
    };
  }, [expanded, maybeCollapse]);

  const goto = useCallback(
    (page: string) => {
      history.push(page);
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    },
    [history]
  );

  const pathname = location.pathname.replace(/\/$/, "");
  let newPath =
    isEditor && newPathsList.includes(pathname) ? `${pathname}/new` : null;
  if (newPath !== null) {
    const queryParam = new URLSearchParams(location.search).get("q");
    if (queryParam) {
      newPath += "?q=" + encodeURIComponent(queryParam);
    }
  }

  // set up hotkeys
  useEffect(() => {
    Mousetrap.bind("?", () => openManual());
    Mousetrap.bind("g z", () => goto("/settings"));

    menuItems.forEach((item) => {
      Mousetrap.bind(item.hotkey, () => goto(item.href));
    });

    if (newPath) {
      Mousetrap.bind("n", () => history.push(String(newPath)));
    }

    return () => {
      Mousetrap.unbind("?");
      Mousetrap.unbind("g z");
      menuItems.forEach((item) => {
        Mousetrap.unbind(item.hotkey);
      });

      if (newPath) {
        Mousetrap.unbind("n");
      }
    };
  });

  function maybeRenderLogout() {
    if (SessionUtils.isLoggedIn()) {
      return (
        <>
          {me?.username && (
            <NavLink
              className="nav-utility stasharr-user-link"
              exact
              to="/settings?tab=account"
              onClick={handleDismiss}
              title={`Signed in as ${me.username}`}
            >
              <Button className="minimal d-flex align-items-center h-100">
                <Icon icon={faUserCircle} />
                <span className="d-none d-md-inline ml-1">{me.username}</span>
              </Button>
            </NavLink>
          )}
          <Button
          className="minimal logout-button d-flex align-items-center"
          href={`${baseURL}logout`}
          title={intl.formatMessage({ id: "actions.logout" })}
        >
          <Icon icon={faSignOutAlt} />
          </Button>
        </>
      );
    }
  }

  const handleDismiss = useCallback(() => setExpanded(false), []);

  function renderUtilityButtons() {
    return (
      <>
        <Nav.Link
          className="nav-utility"
          href="https://opencollective.com/stashapp"
          target="_blank"
          onClick={handleDismiss}
        >
          <Button
            className="minimal donate"
            title={intl.formatMessage({ id: "donate" })}
          >
            <Icon icon={faHeart} />
            <span className="d-none d-sm-inline">
              {intl.formatMessage(messages.donate)}
            </span>
          </Button>
        </Nav.Link>
        {isAdmin && siteOn && (
          <NavLink
            className="nav-utility"
            to="/site"
            onClick={handleDismiss}
          >
            <Button
              className="minimal d-flex align-items-center h-100"
              title="Open the site, as viewers see it"
            >
              <Icon icon={faStore} />
              <span className="d-none d-md-inline ml-1">Site</span>
            </Button>
          </NavLink>
        )}
        <NavLink
          className="nav-utility"
          exact
          to="/stats"
          onClick={handleDismiss}
        >
          <Button
            className="minimal d-flex align-items-center h-100"
            title={intl.formatMessage({ id: "statistics" })}
          >
            <Icon icon={faChartColumn} />
          </Button>
        </NavLink>
        <NavLink
          className="nav-utility"
          exact
          to="/settings"
          onClick={handleDismiss}
        >
          <SettingsButton />
        </NavLink>
        <Button
          className="nav-utility minimal"
          onClick={() => openManual()}
          title={intl.formatMessage({ id: "help" })}
        >
          <Icon icon={faQuestionCircle} />
        </Button>
        {maybeRenderLogout()}
      </>
    );
  }

  return (
    <Navbar
      collapseOnSelect
      fixed="top"
      variant="dark"
      bg="dark"
      className="top-nav"
      expand="xl"
      expanded={expanded}
      onToggle={setExpanded}
      ref={navbarRef}
    >
      <Navbar.Collapse className="bg-dark order-sm-1">
        <MainNavbarMenuItems>
          {menuItems.map(({ href, icon, message }) => (
            <Nav.Link
              eventKey={href}
              as="div"
              key={href}
              className="col-4 col-sm-3 col-md-2 col-lg-auto"
            >
              <LinkContainer activeClassName="active" exact to={href}>
                <Button className="minimal p-4 p-xl-2 d-flex d-xl-inline-block flex-column justify-content-between align-items-center">
                  <Icon
                    {...{ icon }}
                    className="nav-menu-icon d-block d-xl-inline mb-2 mb-xl-0"
                  />
                  <span>{intl.formatMessage(message)}</span>
                </Button>
              </LinkContainer>
            </Nav.Link>
          ))}
        </MainNavbarMenuItems>
        <Nav>
          <MainNavbarUtilityItems>
            {renderUtilityButtons()}
          </MainNavbarUtilityItems>
        </Nav>
      </Navbar.Collapse>

      <Navbar.Brand as="div" onClick={handleDismiss}>
        <Link to="/">
          <Button className="minimal brand-link d-inline-block">Stasharr</Button>
        </Link>
      </Navbar.Brand>

      <Nav className="navbar-buttons flex-row ml-auto order-xl-2">
        {!!newPath && (
          <div className="mr-2">
            <Link to={newPath}>
              <Button variant="primary" data-action="new">
                <FormattedMessage id="new" defaultMessage="New" />
              </Button>
            </Link>
          </div>
        )}
        {/* Stasharr: one tap to Reels and Playlists on phones */}
        {canViewReels && (
          <NavLink
            className="nav-utility d-sm-none"
            exact
            to="/reels"
            onClick={handleDismiss}
          >
            <Button className="minimal d-flex align-items-center h-100" title="Reels">
              <Icon icon={faMobileAlt} />
            </Button>
          </NavLink>
        )}
        <NavLink
          className="nav-utility d-sm-none"
          exact
          to="/playlists"
          onClick={handleDismiss}
        >
          <Button className="minimal d-flex align-items-center h-100" title="Playlists">
            <Icon icon={faListUl} />
          </Button>
        </NavLink>
        <MainNavbarUtilityItems>
          {renderUtilityButtons()}
        </MainNavbarUtilityItems>
        <Navbar.Toggle className="nav-menu-toggle ml-sm-2">
          <Icon icon={expanded ? faTimes : faBars} />
        </Navbar.Toggle>
      </Nav>
    </Navbar>
  );
};
