import { useEffect } from "react";
import * as GQL from "src/core/generated-graphql";

export interface ICurrentUser {
  loading: boolean;
  me?: GQL.CurrentUserDataFragment | null;
  isAdmin: boolean;
  isEditor: boolean;
  isViewer: boolean;
  canViewReels: boolean;
  canDownload: boolean;
}

// The role of the logged in account. With no login configured, or while the
// query is loading, full access is assumed so the UI matches upstream Stash.
// The server enforces the real permissions either way.
export function useCurrentUser(): ICurrentUser {
  const { data, loading } = GQL.useMeQuery({ fetchPolicy: "cache-first" });
  const me = data?.me;
  const role = me?.role ?? GQL.UserRole.Admin;

  const isAdmin = role === GQL.UserRole.Admin;
  const isEditor = isAdmin || role === GQL.UserRole.Editor;

  return {
    loading,
    me,
    isAdmin,
    isEditor,
    isViewer: role === GQL.UserRole.Viewer,
    canViewReels: me?.can_view_reels ?? true,
    canDownload: me?.can_download ?? true,
  };
}

// Adds a stasharr-role-* class to the page body so styles can hide controls
// a role cannot use.
export function useRoleBodyClass() {
  const { me } = useCurrentUser();
  const role = (me?.role ?? GQL.UserRole.Admin).toLowerCase();
  const noDownload = me ? !me.can_download : false;

  useEffect(() => {
    const cls = `stasharr-role-${role}`;
    document.body.classList.add(cls);
    if (noDownload) {
      document.body.classList.add("stasharr-no-download");
    }
    return () => {
      document.body.classList.remove(cls);
      document.body.classList.remove("stasharr-no-download");
    };
  }, [role, noDownload]);
}
