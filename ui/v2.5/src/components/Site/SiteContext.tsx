import React, { useContext } from "react";
import * as GQL from "src/core/generated-graphql";
import { ISiteStyle, findSiteStyle } from "./siteStyles";

export interface ISiteContext {
  config: GQL.SiteConfigDataFragment;
  style: ISiteStyle;
}

const defaultConfig: GQL.SiteConfigDataFragment = {
  __typename: "SiteConfig",
  enabled: false,
  audience: "viewers",
  style: "velvet",
  name: "Stasharr",
  tagline: "",
  featured: "top_rated",
  featured_tag: null,
  models_gender: "FEMALE",
  applies_to_me: false,
  home_rows: [],
  promo: { __typename: "SitePromo", enabled: false, text: "", link: "", image: "" },
  age_gate: false,
  accent: "",
  logo_url: null,
  app_look: true,
};

export const SiteContext = React.createContext<ISiteContext>({
  config: defaultConfig,
  style: findSiteStyle("velvet"),
});

export function useSite(): ISiteContext {
  return useContext(SiteContext);
}

// The site settings, for the redirect in App and the admin menu.
export function useSiteConfig() {
  const { data, loading } = GQL.useSiteConfigQuery({ fetchPolicy: "cache-first" });
  return { config: data?.siteConfig, loading };
}
