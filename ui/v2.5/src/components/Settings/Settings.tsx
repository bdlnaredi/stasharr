import React from "react";
import { Tab, Nav, Row, Col, Form } from "react-bootstrap";
import { Redirect, useLocation } from "react-router-dom";
import { LinkContainer } from "react-router-bootstrap";
import { FormattedMessage } from "react-intl";
import { Helmet } from "react-helmet";
import { useTitleProps } from "src/hooks/title";
import { SettingsAboutPanel } from "./SettingsAboutPanel";
import { SettingsConfigurationPanel } from "./SettingsSystemPanel";
import { SettingsInterfacePanel } from "./SettingsInterfacePanel/SettingsInterfacePanel";
import { SettingsLogsPanel } from "./SettingsLogsPanel";
import { SettingsTasksPanel } from "./Tasks/SettingsTasksPanel";
import { SettingsPluginsPanel } from "./SettingsPluginsPanel";
import { SettingsScrapingPanel } from "./SettingsScrapingPanel";
import { SettingsToolsPanel } from "./SettingsToolsPanel";
import { SettingsServicesPanel } from "./SettingsServicesPanel";
import { SettingsContext, useSettings } from "./context";
import { SettingsLibraryPanel } from "./SettingsLibraryPanel";
import { SettingsSecurityPanel } from "./SettingsSecurityPanel";
import Changelog from "../Changelog/Changelog";
import { useCurrentUser } from "../Stasharr/CurrentUser";
import { UsersPanel } from "../Stasharr/UsersPanel";
import { AutoImportPanel } from "../Stasharr/AutoImportPanel";
import { AccountPanel } from "../Stasharr/AccountPanel";
import { TroubleshootingModeButton } from "../TroubleshootingMode/TroubleshootingModeButton";
import { useTroubleshootingMode } from "../TroubleshootingMode/useTroubleshootingMode";

const validTabs = [
  "tasks",
  "library",
  "interface",
  "security",
  "users",
  "auto-import",
  "account",
  "metadata-providers",
  "services",
  "system",
  "plugins",
  "logs",
  "tools",
  "changelog",
  "about",
] as const;
type TabKey = (typeof validTabs)[number];

const defaultTab: TabKey = "tasks";

interface ISettingsTab {
  key: TabKey;
  label: React.ReactNode;
}

const upstreamTabs: ISettingsTab[] = [
  {
    key: "tasks",
    label: <FormattedMessage id="config.categories.tasks" />,
  },
  {
    key: "library",
    label: <FormattedMessage id="library" />,
  },
  {
    key: "interface",
    label: <FormattedMessage id="config.categories.interface" />,
  },
  {
    key: "security",
    label: <FormattedMessage id="config.categories.security" />,
  },
  {
    key: "metadata-providers",
    label: <FormattedMessage id="config.categories.metadata_providers" />,
  },
  {
    key: "services",
    label: <FormattedMessage id="config.categories.services" />,
  },
  {
    key: "system",
    label: <FormattedMessage id="config.categories.system" />,
  },
  {
    key: "plugins",
    label: <FormattedMessage id="config.categories.plugins" />,
  },
  {
    key: "logs",
    label: <FormattedMessage id="config.categories.logs" />,
  },
  {
    key: "tools",
    label: <FormattedMessage id="config.categories.tools" />,
  },
  {
    key: "changelog",
    label: <FormattedMessage id="config.categories.changelog" />,
  },
  {
    key: "about",
    label: <FormattedMessage id="config.categories.about" />,
  },
];

// tabs each role can open
const editorTabs: TabKey[] = [
  "tasks",
  "auto-import",
  "account",
  "changelog",
  "about",
];
const viewerTabs: TabKey[] = ["account", "about"];

function tabsForRole(isAdmin: boolean, isEditor: boolean): ISettingsTab[] {
  const extra: ISettingsTab[] = [
    { key: "users", label: "Users & tiers" },
    { key: "auto-import", label: "Auto-import" },
    { key: "account", label: "Account" },
  ];
  // new tabs go after Security
  const all: ISettingsTab[] = [];
  upstreamTabs.forEach((t) => {
    all.push(t);
    if (t.key === "security") all.push(...extra);
  });
  if (isAdmin) return all;
  const allowed = isEditor ? editorTabs : viewerTabs;
  return all.filter((t) => allowed.includes(t.key));
}

function isTabKey(tab: string | null): tab is TabKey {
  return validTabs.includes(tab as TabKey);
}

const SettingTabs: React.FC<{ tab: TabKey }> = ({ tab }) => {
  const { advancedMode, setAdvancedMode } = useSettings();
  const { isAdmin, isEditor } = useCurrentUser();
  const tabs = tabsForRole(isAdmin, isEditor);
  const { isActive: troubleshootingModeActive } = useTroubleshootingMode();

  const titleProps = useTitleProps({ id: "settings" });

  return (
    <Tab.Container activeKey={tab} id="configuration-tabs">
      <Helmet {...titleProps} />
      <Row>
        <Col id="settings-menu-container" sm={3} md={3} xl={2}>
          <Nav variant="pills" className="flex-column">
            {tabs.map((t) => (
              <Nav.Item key={t.key}>
                <LinkContainer to={`/settings?tab=${t.key}`}>
                  <Nav.Link eventKey={t.key}>{t.label}</Nav.Link>
                </LinkContainer>
              </Nav.Item>
            ))}
            <Nav.Item className={isAdmin ? "" : "d-none"}>
              <div className="advanced-switch">
                <Form.Label htmlFor="advanced-settings">
                  <FormattedMessage id="config.advanced_mode" />
                </Form.Label>
                <Form.Switch
                  id="advanced-settings"
                  checked={advancedMode}
                  onChange={() => setAdvancedMode(!advancedMode)}
                />
              </div>
            </Nav.Item>
            {isAdmin && !troubleshootingModeActive && (
              <TroubleshootingModeButton />
            )}
            <hr className="d-sm-none" />
          </Nav>
        </Col>
        <Col
          id="settings-container"
          sm={{ offset: 3 }}
          md={{ offset: 3 }}
          xl={{ offset: 2 }}
        >
          <Tab.Content className="mx-auto">
            <Tab.Pane eventKey="library">
              <SettingsLibraryPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="interface">
              <SettingsInterfacePanel />
            </Tab.Pane>
            <Tab.Pane eventKey="security">
              <SettingsSecurityPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="users" unmountOnExit>
              <UsersPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="auto-import" unmountOnExit>
              <AutoImportPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="account" unmountOnExit>
              <AccountPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="tasks">
              <SettingsTasksPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="services" unmountOnExit>
              <SettingsServicesPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="tools" unmountOnExit>
              <SettingsToolsPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="metadata-providers" unmountOnExit>
              <SettingsScrapingPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="system">
              <SettingsConfigurationPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="plugins" unmountOnExit>
              <SettingsPluginsPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="logs" unmountOnExit>
              <SettingsLogsPanel />
            </Tab.Pane>
            <Tab.Pane eventKey="changelog" unmountOnExit>
              <Changelog />
            </Tab.Pane>
            <Tab.Pane eventKey="about" unmountOnExit>
              <SettingsAboutPanel />
            </Tab.Pane>
          </Tab.Content>
        </Col>
      </Row>
    </Tab.Container>
  );
};

export const Settings: React.FC = () => {
  const location = useLocation();
  const tab = new URLSearchParams(location.search).get("tab");
  const { isAdmin, isEditor, loading } = useCurrentUser();

  if (loading) {
    return null;
  }

  const allowed = tabsForRole(isAdmin, isEditor).map((t) => t.key);
  if (!isTabKey(tab) || !allowed.includes(tab)) {
    const fallback = allowed.includes(defaultTab) ? defaultTab : allowed[0];
    return (
      <Redirect
        to={{
          ...location,
          search: `tab=${fallback}`,
        }}
      />
    );
  }

  return (
    <SettingsContext>
      <SettingTabs tab={tab} />
    </SettingsContext>
  );
};

export default Settings;
