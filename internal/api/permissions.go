package api

// Role based access to the GraphQL API.
//
// Every root Query, Mutation and Subscription field is checked against the
// role of the current user before it runs. Requests without a principal
// (no login configured, or internal plugin calls) are not restricted.

import (
	"context"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"

	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
)

// Mutations a viewer may run: their own play/O history, ratings and password.
var viewerMutations = map[string]bool{
	"sceneUpdate":             true, // rating only, checked below
	"sceneIncrementO":         true,
	"sceneDecrementO":         true,
	"sceneAddO":               true,
	"sceneDeleteO":            true,
	"sceneResetO":             true,
	"sceneSaveActivity":       true,
	"sceneResetActivity":      true,
	"sceneIncrementPlayCount": true,
	"sceneAddPlay":            true,
	"sceneDeletePlay":         true,
	"sceneResetPlayCount":     true,
	"changeMyPassword":        true,
	// own playlists
	"playlistCreate":       true,
	"playlistUpdate":       true,
	"playlistDestroy":      true,
	"playlistAddScenes":    true,
	"playlistRemoveScenes": true,
	"playlistReorder":      true,
	"watchLaterToggle":     true,
	// accepted but ignored for non-admins, see ConfigureUI
	"configureUI":        true,
	"configureUISetting": true,
}

// Mutations only an admin may run. Everything else is open to editors.
var adminMutations = map[string]bool{
	"setup": true, "migrate": true, "downloadFFMpeg": true,
	"moveFiles": true, "deleteFiles": true, "destroyFiles": true,
	"revealFileInFileManager": true, "revealFolderInFileManager": true,
	"configureGeneral": true, "configureInterface": true, "configureDLNA": true,
	"configureScraping": true, "configureDefaults": true, "configurePlugin": true,
	"generateAPIKey": true,
	"exportObjects":  true, "importObjects": true, "metadataImport": true, "metadataExport": true,
	"metadataClean": true, "metadataCleanGenerated": true,
	"migrateHashNaming": true, "migrateSceneScreenshots": true, "migrateBlobs": true,
	"anonymiseDatabase": true, "optimiseDatabase": true, "backupDatabase": true,
	"querySQL": true, "execSQL": true,
	"reloadScrapers": true, "setPluginsEnabled": true, "runPluginTask": true, "runPluginOperation": true,
	"reloadPlugins": true, "installPackages": true, "updatePackages": true, "uninstallPackages": true,
	"enableDLNA": true, "disableDLNA": true, "addTempDLNAIP": true, "removeTempDLNAIP": true,
	"configureLibraryWatcher": true,
	"userCreate":              true, "userUpdate": true, "userDestroy": true, "userGenerateAPIKey": true,
	"userTierCreate": true, "userTierUpdate": true, "userTierDestroy": true,
	"configureTrash": true, "trashRestore": true, "trashDelete": true,
	"duplicateCleanup": true, "configureRenamer": true, "renameScenes": true,
}

// Mutations that delete files from disk when delete_file is set.
var fileDeletingMutations = map[string]bool{
	"sceneDestroy": true, "scenesDestroy": true,
	"imageDestroy": true, "imagesDestroy": true,
	"galleryDestroy": true,
}

// Queries hidden from viewers.
var editorQueries = map[string]bool{
	"directory": true, "parseSceneFilenames": true, "findDuplicateScenes": true,
	"validateStashBoxCredentials": true,
	"findScenesByPathRegex":       true, "libraryWatcherConfig": true,
	"scrapeSingleScene": true, "scrapeMultiScenes": true, "scrapeSingleStudio": true,
	"scrapeSingleTag": true, "scrapeSinglePerformer": true, "scrapeMultiPerformers": true,
	"scrapeSingleGallery": true, "scrapeSingleMovie": true, "scrapeSingleGroup": true,
	"scrapeSingleImage": true, "scrapeURL": true, "scrapePerformerURL": true,
	"scrapeSceneURL": true, "scrapeGalleryURL": true, "scrapeImageURL": true,
	"scrapeMovieURL": true, "scrapeGroupURL": true,
}

// Queries and subscriptions only for admins.
var adminQueries = map[string]bool{
	"logs": true, "installedPackages": true, "availablePackages": true,
	"dlnaStatus": true, "users": true, "userTiers": true,
	"loggingSubscribe": true,
	"trashBin":         true, "duplicateCleanupPreview": true, "renamerConfig": true, "renamePreview": true,
}

func requiredRole(op string, field string) access.Role {
	switch op {
	case "Mutation":
		if viewerMutations[field] {
			return access.RoleViewer
		}
		if adminMutations[field] {
			return access.RoleAdmin
		}
		return access.RoleEditor
	default:
		if adminQueries[field] {
			return access.RoleAdmin
		}
		if editorQueries[field] {
			return access.RoleEditor
		}
		return access.RoleViewer
	}
}

// permissionFieldMiddleware enforces roles on root fields.
func permissionFieldMiddleware(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || (fc.Object != "Query" && fc.Object != "Mutation" && fc.Object != "Subscription") {
		// not a root field
		return next(ctx)
	}

	p := access.FromContext(ctx)
	if p == nil || p.Role == access.RoleAdmin {
		return next(ctx)
	}

	op := fc.Object
	field := fc.Field.Name
	if field == "__typename" || strings.HasPrefix(field, "__") {
		return next(ctx)
	}

	if !p.Role.AtLeast(requiredRole(op, field)) {
		return nil, fmt.Errorf("%w (%s)", access.ErrForbidden, field)
	}

	if op == "Mutation" {
		input := getUpdateInputMap(ctx)
		if field == "sceneUpdate" && p.Role == access.RoleViewer {
			for k := range input {
				if k != "id" && k != "rating100" {
					return nil, fmt.Errorf("%w: viewers can only rate scenes", access.ErrForbidden)
				}
			}
		}
		if fileDeletingMutations[field] {
			if del, _ := input["delete_file"].(bool); del {
				return nil, fmt.Errorf("%w: only admins can delete files from disk", access.ErrForbidden)
			}
		}
	}

	return next(ctx)
}

// redactConfig hides secrets and other users' library paths from non-admins.
func redactConfig(ctx context.Context, c *ConfigResult) *ConfigResult {
	p := access.FromContext(ctx)
	if p == nil || p.Role == access.RoleAdmin || c == nil {
		return c
	}
	ret := *c
	if c.General != nil {
		g := *c.General
		g.APIKey = ""
		g.Password = ""
		boxes := make([]*models.StashBox, 0, len(g.StashBoxes))
		for _, b := range g.StashBoxes {
			cp := *b
			cp.APIKey = ""
			boxes = append(boxes, &cp)
		}
		g.StashBoxes = boxes
		if p.RestrictsContent() {
			g.Stashes = nil
			for _, s := range c.General.Stashes {
				if p.Policy.PathAllowed(s.Path) || containsAllowedPath(p.Policy, s.Path) {
					g.Stashes = append(g.Stashes, s)
				}
			}
		}
		ret.General = &g
	}
	if p.Role == access.RoleViewer {
		ret.Plugins = map[string]map[string]interface{}{}
	}
	return &ret
}

// containsAllowedPath reports whether a library folder contains one of the
// tier's allowed sub-folders.
func containsAllowedPath(p access.Policy, libraryPath string) bool {
	for _, a := range p.AllowedPaths {
		if (access.Policy{AllowedPaths: []string{libraryPath}}).PathAllowed(a) {
			return true
		}
	}
	return false
}
