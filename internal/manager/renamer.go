package manager

// Stasharr file renamer: moves scene files into a folder and file name built
// from a template, such as "{studio}/{date} - {title}". Files stay inside the
// library folder they are already in. Subtitles and funscripts with the same
// name are moved along with the video.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
	"github.com/stashapp/stash/pkg/txn"
)

// Rename statuses.
const (
	RenameMove = "MOVE"
	RenameSame = "SAME"
	RenameSkip = "SKIP"
)

// RenamerTokens lists the template tokens and what they are replaced with.
var RenamerTokens = [][2]string{
	{"{title}", "Scene title"},
	{"{studio}", "Studio name"},
	{"{parent_studio}", "Parent studio (network) name, or the studio when it has none"},
	{"{date}", "Release date, YYYY-MM-DD"},
	{"{year}", "Release year"},
	{"{month}", "Release month, 01-12"},
	{"{performers}", "Up to three performers, comma separated"},
	{"{performer}", "First performer"},
	{"{code}", "Studio code"},
	{"{director}", "Director"},
	{"{resolution}", "Video resolution, such as 1080p or 2160p"},
	{"{codec}", "Video codec, such as H264 or HEVC"},
	{"{stash_id}", "First stash-box id"},
	{"{id}", "Stasharr scene id"},
}

// RenamePlan is the planned rename of one scene file.
type RenamePlan struct {
	SceneID int
	FileID  models.FileID
	Title   string
	OldPath string
	NewPath string
	Status  string
	Reason  string
}

var (
	tokenRe        = regexp.MustCompile(`\{[a-z_]+\}`)
	illegalRe      = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	emptyBracketRe = regexp.MustCompile(`\(\s*\)|\[\s*\]|\{\s*\}`)
	spacesRe       = regexp.MustCompile(`\s{2,}`)
	// separators left behind by empty tokens, such as " -  - "
	repeatedSepRe = regexp.MustCompile(`(\s*[-–_.,]\s*){2,}`)
)

const maxNameBytes = 200

func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for !utf8.ValidString(s) && len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s
}

func cleanSegment(s string) string {
	s = illegalRe.ReplaceAllString(s, "")
	for i := 0; i < 3; i++ {
		s = emptyBracketRe.ReplaceAllString(s, "")
	}
	s = repeatedSepRe.ReplaceAllStringFunc(s, func(m string) string {
		t := strings.TrimSpace(m)
		if t == "" {
			return " "
		}
		return " " + string([]rune(t)[0]) + " "
	})
	s = spacesRe.ReplaceAllString(s, " ")
	s = strings.Trim(s, " -–_.,")
	return truncateBytes(s, maxNameBytes)
}

type renameContext struct {
	scene      *models.Scene
	file       *models.VideoFile
	studio     *models.Studio
	parent     *models.Studio
	performers []string
}

func (c renameContext) value(token string) string {
	s := c.scene
	switch token {
	case "{title}":
		return s.Title
	case "{studio}":
		if c.studio != nil {
			return c.studio.Name
		}
	case "{parent_studio}":
		if c.parent != nil {
			return c.parent.Name
		}
		if c.studio != nil {
			return c.studio.Name
		}
	case "{date}":
		if s.Date != nil {
			return s.Date.String()
		}
	case "{year}":
		if s.Date != nil {
			return s.Date.Format("2006")
		}
	case "{month}":
		if s.Date != nil && s.Date.Precision <= models.DatePrecisionMonth {
			return s.Date.Format("01")
		}
	case "{performers}":
		n := len(c.performers)
		if n > 3 {
			n = 3
		}
		return strings.Join(c.performers[:n], ", ")
	case "{performer}":
		if len(c.performers) > 0 {
			return c.performers[0]
		}
	case "{code}":
		return s.Code
	case "{director}":
		return s.Director
	case "{resolution}":
		if c.file != nil && c.file.Width > 0 && c.file.Height > 0 {
			short := c.file.Height
			if c.file.Width < short {
				short = c.file.Width
			}
			return strconv.Itoa(short) + "p"
		}
	case "{codec}":
		if c.file != nil {
			return strings.ToUpper(c.file.VideoCodec)
		}
	case "{stash_id}":
		if ids := s.StashIDs.List(); len(ids) > 0 {
			return ids[0].StashID
		}
	case "{id}":
		return strconv.Itoa(s.ID)
	}
	return ""
}

// renderTemplate returns the path relative to the library folder, without
// the extension.
func renderTemplate(tmpl string, c renameContext) string {
	tmpl = strings.ReplaceAll(tmpl, `\`, "/")
	var parts []string
	for _, seg := range strings.Split(tmpl, "/") {
		out := tokenRe.ReplaceAllStringFunc(seg, func(t string) string {
			// a folder separator inside a value must not create folders
			return strings.NewReplacer("/", " ", `\`, " ").Replace(c.value(t))
		})
		out = cleanSegment(out)
		if out != "" {
			parts = append(parts, out)
		}
	}
	return strings.Join(parts, "/")
}

// ValidateRenamerTemplate checks a template for unknown tokens.
func ValidateRenamerTemplate(tmpl string) error {
	if strings.TrimSpace(tmpl) == "" {
		return errors.New("the template is empty")
	}
	known := map[string]bool{}
	for _, t := range RenamerTokens {
		known[t[0]] = true
	}
	for _, t := range tokenRe.FindAllString(tmpl, -1) {
		if !known[t] {
			return fmt.Errorf("unknown token %s", t)
		}
	}
	if !tokenRe.MatchString(tmpl) {
		return errors.New("the template has no tokens, so every file would get the same name")
	}
	if strings.Contains(tmpl, "..") {
		return errors.New("the template may not contain \"..\"")
	}
	return nil
}

func loadRenameContext(ctx context.Context, r models.Repository, s *models.Scene) (renameContext, error) {
	c := renameContext{scene: s}
	if err := s.LoadFiles(ctx, r.Scene); err != nil {
		return c, err
	}
	if err := s.LoadStashIDs(ctx, r.Scene); err != nil {
		return c, err
	}
	if err := s.LoadPerformerIDs(ctx, r.Scene); err != nil {
		return c, err
	}
	c.file = primaryVideo(s)
	if s.StudioID != nil {
		st, err := r.Studio.Find(ctx, *s.StudioID)
		if err != nil {
			return c, err
		}
		c.studio = st
		if st != nil && st.ParentID != nil {
			p, err := r.Studio.Find(ctx, *st.ParentID)
			if err != nil {
				return c, err
			}
			c.parent = p
		}
	}
	if ids := s.PerformerIDs.List(); len(ids) > 0 {
		ps, err := r.Performer.FindMany(ctx, ids)
		if err != nil {
			return c, err
		}
		for _, p := range ps {
			if p != nil && p.Name != "" {
				c.performers = append(c.performers, p.Name)
			}
		}
		sort.Strings(c.performers)
	}
	return c, nil
}

func pathExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// planRename works out the new path of one scene. taken holds paths already
// planned for other scenes in the same run.
func planRename(ctx context.Context, r models.Repository, cfg *config.Config, s *models.Scene, tmpl string, taken map[string]bool) (RenamePlan, error) {
	plan := RenamePlan{SceneID: s.ID, Title: s.GetTitle(), Status: RenameSkip}
	c, err := loadRenameContext(ctx, r, s)
	if err != nil {
		return plan, err
	}
	if c.file == nil {
		plan.Reason = "No file"
		return plan, nil
	}
	plan.FileID = c.file.ID
	plan.OldPath = c.file.Path

	if c.file.ZipFileID != nil {
		plan.Reason = "File is inside a zip file"
		return plan, nil
	}
	if strings.Contains(tmpl, "{title}") && s.Title == "" {
		plan.Reason = "No title yet: identify the scene first"
		return plan, nil
	}

	stash := cfg.GetStashPaths().GetStashFromPath(c.file.Path)
	if stash == nil {
		plan.Reason = "Not in a library folder"
		return plan, nil
	}

	rel := renderTemplate(tmpl, c)
	if rel == "" {
		plan.Reason = "The template gives an empty name for this scene"
		return plan, nil
	}

	ext := strings.ToLower(filepath.Ext(c.file.Path))
	base := filepath.Join(stash.Path, filepath.FromSlash(rel))
	newPath := base + ext
	if newPath == c.file.Path {
		plan.NewPath = newPath
		plan.Status = RenameSame
		plan.Reason = "Already named correctly"
		return plan, nil
	}

	for n := 2; (pathExists(newPath) || taken[newPath]) && !strings.EqualFold(newPath, c.file.Path); n++ {
		if n > 9 {
			plan.Reason = "A file with this name already exists"
			plan.NewPath = base + ext
			return plan, nil
		}
		newPath = fmt.Sprintf("%s (%d)%s", base, n, ext)
		if newPath == c.file.Path {
			plan.NewPath = newPath
			plan.Status = RenameSame
			plan.Reason = "Already named correctly"
			return plan, nil
		}
	}

	plan.NewPath = newPath
	plan.Status = RenameMove
	taken[newPath] = true
	return plan, nil
}

// PlanRenames works out the new paths of the given scenes, or of every scene
// when ids is empty. limit caps the number of scenes looked at (0 = no cap).
func PlanRenames(ctx context.Context, ids []int, tmpl string, limit int) ([]RenamePlan, error) {
	mgr := GetInstance()
	r := mgr.Repository
	if tmpl == "" {
		tmpl = mgr.Config.GetRenamerTemplate()
	}
	if err := ValidateRenamerTemplate(tmpl); err != nil {
		return nil, err
	}

	var ret []RenamePlan
	taken := map[string]bool{}
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		var scenes []*models.Scene
		var err error
		if len(ids) > 0 {
			scenes, err = r.Scene.FindMany(ctx, ids)
		} else {
			scenes, err = r.Scene.All(ctx)
		}
		if err != nil {
			return err
		}
		for _, s := range scenes {
			if s == nil {
				continue
			}
			if limit > 0 && len(ret) >= limit {
				break
			}
			p, err := planRename(ctx, r, mgr.Config, s, tmpl, taken)
			if err != nil {
				return err
			}
			ret = append(ret, p)
		}
		return nil
	})
	return ret, err
}

// sidecars returns files next to the video with the same name and another
// extension, such as subtitles and funscripts.
func sidecars(videoPath string) []string {
	dir := filepath.Dir(videoPath)
	stem := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var ret []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == filepath.Base(videoPath) {
			continue
		}
		if strings.HasPrefix(name, stem+".") && !useAsVideo(filepath.Join(dir, name)) {
			ret = append(ret, filepath.Join(dir, name))
		}
	}
	return ret
}

// removeEmptyParents removes empty folders left behind after a move, up to
// but not including the library folder.
func removeEmptyParents(dir string, root string) {
	root = filepath.Clean(root)
	for {
		dir = filepath.Clean(dir)
		if dir == root || !strings.HasPrefix(dir, root+string(filepath.Separator)) {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

// applyRename moves one file and its sidecars.
func applyRename(ctx context.Context, plan RenamePlan) error {
	mgr := GetInstance()
	r := mgr.Repository
	roots := mgr.Config.GetStashPaths().Paths()
	stash := mgr.Config.GetStashPaths().GetStashFromPath(plan.OldPath)

	extras := sidecars(plan.OldPath)
	oldStem := strings.TrimSuffix(filepath.Base(plan.OldPath), filepath.Ext(plan.OldPath))
	newStem := strings.TrimSuffix(filepath.Base(plan.NewPath), filepath.Ext(plan.NewPath))
	newDir := filepath.Dir(plan.NewPath)

	var movedExtras [][2]string
	err := r.WithTxn(ctx, func(ctx context.Context) error {
		files, err := r.File.Find(ctx, plan.FileID)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return fmt.Errorf("file %s is no longer in the library", plan.OldPath)
		}
		f := files[0]
		if f.Base().Path != plan.OldPath {
			return fmt.Errorf("file %s has moved since the preview", plan.OldPath)
		}

		mover := file.NewMover(r.File, r.Folder, roots)
		mover.RegisterHooks(ctx)

		folder, err := file.GetOrCreateFolderHierarchy(ctx, r.Folder, newDir, roots)
		if err != nil {
			return fmt.Errorf("creating folder %s: %w", newDir, err)
		}
		if err := mover.CreateFolderHierarchy(folder.Path); err != nil {
			return fmt.Errorf("creating folder %s: %w", folder.Path, err)
		}
		if err := mover.Move(ctx, f, folder, filepath.Base(plan.NewPath)); err != nil {
			return err
		}

		if err := sqlite.RenameCaptions(ctx, plan.FileID, oldStem, newStem); err != nil {
			return err
		}

		// move sidecars; put them back if the transaction fails
		txn.AddPostRollbackHook(ctx, func(ctx context.Context) {
			for _, m := range movedExtras {
				_ = os.Rename(m[1], m[0])
			}
		})
		for _, src := range extras {
			dst := filepath.Join(newDir, newStem+strings.TrimPrefix(filepath.Base(src), oldStem))
			if pathExists(dst) {
				logger.Warnf("[renamer] not moving %s: %s already exists", src, dst)
				continue
			}
			if err := os.Rename(src, dst); err != nil {
				logger.Warnf("[renamer] could not move %s: %v", src, err)
				continue
			}
			movedExtras = append(movedExtras, [2]string{src, dst})
		}
		return nil
	})
	if err != nil {
		return err
	}

	if stash != nil {
		removeEmptyParents(filepath.Dir(plan.OldPath), stash.Path)
	}
	logger.Infof("[renamer] %s -> %s", plan.OldPath, plan.NewPath)
	return nil
}

// RenameJob renames scene files using the renamer template.
type RenameJob struct {
	// SceneIDs to rename. Empty renames every scene.
	SceneIDs []int
	// Template overrides the saved template.
	Template string
}

func (j *RenameJob) Execute(ctx context.Context, progress *job.Progress) error {
	plans, err := PlanRenames(ctx, j.SceneIDs, j.Template, 0)
	if err != nil {
		return err
	}
	var todo []RenamePlan
	for _, p := range plans {
		if p.Status == RenameMove {
			todo = append(todo, p)
		}
	}
	if len(todo) == 0 {
		logger.Info("[renamer] nothing to rename")
		return nil
	}

	progress.SetTotal(len(todo))
	done := 0
	for _, p := range todo {
		if job.IsCancelled(ctx) {
			logger.Info("[renamer] stopped")
			break
		}
		progress.ExecuteTask("Renaming "+filepath.Base(p.OldPath), func() {
			if err := applyRename(ctx, p); err != nil {
				logger.Errorf("[renamer] %s: %v", p.OldPath, err)
				return
			}
			done++
		})
		progress.Increment()
	}
	logger.Infof("[renamer] renamed %d of %d file(s)", done, len(todo))
	return nil
}
