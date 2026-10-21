package manager

// Stasharr duplicate cleanup: in every group of duplicate scenes, keep the
// best copy (highest resolution, then bitrate, then file size), copy the
// metadata and history of the others onto it, and send the other files to
// the trash bin.

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene"
	"github.com/stashapp/stash/pkg/sqlite"
)

// DuplicateCleanupOptions selects which scenes count as duplicates.
type DuplicateCleanupOptions struct {
	// Distance is the phash distance: 0 exact, 4 high, 8 medium, 10 low.
	Distance int
	// DurationDiff is the largest difference in length, in seconds. A
	// negative value ignores the length.
	DurationDiff float64
	// KeepSceneIDs limits the cleanup to the groups whose best scene is one
	// of these. Empty means every group.
	KeepSceneIDs []int
}

// DuplicateGroupPlan is one group of duplicates: the scene to keep and the
// scenes to remove.
type DuplicateGroupPlan struct {
	Keep   *models.Scene
	Remove []*models.Scene
}

func primaryVideo(s *models.Scene) *models.VideoFile {
	if f := s.Files.Primary(); f != nil {
		return f
	}
	if l := s.Files.List(); len(l) > 0 {
		return l[0]
	}
	return nil
}

// betterScene reports whether a is a better copy to keep than b.
func betterScene(a, b *models.Scene) bool {
	fa, fb := primaryVideo(a), primaryVideo(b)
	if fa != nil && fb != nil {
		pa, pb := fa.Width*fa.Height, fb.Width*fb.Height
		if pa != pb {
			return pa > pb
		}
		if fa.BitRate != fb.BitRate {
			return fa.BitRate > fb.BitRate
		}
		if fa.Size != fb.Size {
			return fa.Size > fb.Size
		}
	}
	if a.Organized != b.Organized {
		return a.Organized
	}
	sa, sb := len(a.StashIDs.List()), len(b.StashIDs.List())
	if sa != sb {
		return sa > sb
	}
	return a.ID < b.ID
}

// PlanDuplicateCleanup finds duplicate groups and picks the scene to keep in
// each. It does not change anything.
func PlanDuplicateCleanup(ctx context.Context, opts DuplicateCleanupOptions) ([]DuplicateGroupPlan, error) {
	r := GetInstance().Repository
	var ret []DuplicateGroupPlan
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		groups, err := r.Scene.FindDuplicates(ctx, opts.Distance, opts.DurationDiff, nil)
		if err != nil {
			return err
		}
		for _, g := range groups {
			var scenes []*models.Scene
			for _, s := range g {
				if err := s.LoadFiles(ctx, r.Scene); err != nil {
					return err
				}
				if err := s.LoadStashIDs(ctx, r.Scene); err != nil {
					return err
				}
				if primaryVideo(s) == nil {
					continue
				}
				scenes = append(scenes, s)
			}
			if len(scenes) < 2 {
				continue
			}
			sort.SliceStable(scenes, func(i, j int) bool { return betterScene(scenes[i], scenes[j]) })
			if len(opts.KeepSceneIDs) > 0 && !slices.Contains(opts.KeepSceneIDs, scenes[0].ID) {
				continue
			}
			ret = append(ret, DuplicateGroupPlan{Keep: scenes[0], Remove: scenes[1:]})
		}
		return nil
	})
	return ret, err
}

// DuplicateCleanupJob removes duplicates as planned by PlanDuplicateCleanup.
type DuplicateCleanupJob struct {
	Options DuplicateCleanupOptions
}

func (j *DuplicateCleanupJob) Execute(ctx context.Context, progress *job.Progress) error {
	plans, err := PlanDuplicateCleanup(ctx, j.Options)
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		logger.Info("[duplicates] no duplicates found")
		return nil
	}

	progress.SetTotal(len(plans))
	removed := 0
	for _, p := range plans {
		if job.IsCancelled(ctx) {
			logger.Info("[duplicates] stopped")
			break
		}
		progress.ExecuteTask(fmt.Sprintf("Cleaning up duplicates of %s", p.Keep.GetTitle()), func() {
			n, err := cleanupDuplicateGroup(ctx, p)
			if err != nil {
				logger.Errorf("[duplicates] %s: %v", p.Keep.GetTitle(), err)
			}
			removed += n
		})
		progress.Increment()
	}
	logger.Infof("[duplicates] removed %d duplicate scene(s) in %d group(s); the files are in the trash bin", removed, len(plans))
	return nil
}

func missingIDs(have, add []int) []int {
	var ret []int
	for _, id := range add {
		if !slices.Contains(have, id) && !slices.Contains(ret, id) {
			ret = append(ret, id)
		}
	}
	return ret
}

func cleanupDuplicateGroup(ctx context.Context, plan DuplicateGroupPlan) (int, error) {
	mgr := GetInstance()
	r := mgr.Repository

	fileDeleter := &scene.FileDeleter{
		Deleter:        file.NewDeleterWithTrash(mgr.Config.GetDeleteTrashPath()),
		FileNamingAlgo: mgr.Config.GetVideoFileNamingAlgorithm(),
		Paths:          mgr.Paths,
	}

	removed := 0
	if err := r.WithTxn(ctx, func(ctx context.Context) error {
		keep, err := r.Scene.Find(ctx, plan.Keep.ID)
		if err != nil {
			return err
		}
		if keep == nil {
			return fmt.Errorf("scene %d no longer exists", plan.Keep.ID)
		}
		if err := keep.LoadRelationships(ctx, r.Scene); err != nil {
			return err
		}

		partial := models.NewScenePartial()
		var tagIDs, performerIDs, galleryIDs []int
		var groups []models.GroupsScenes
		var urls []string
		stashIDs := &models.UpdateStashIDs{StashIDs: keep.StashIDs.List(), Mode: models.RelationshipUpdateModeSet}
		stashIDsChanged := false
		organized := keep.Organized

		var dupes []*models.Scene
		for _, d := range plan.Remove {
			s, err := r.Scene.Find(ctx, d.ID)
			if err != nil {
				return err
			}
			if s == nil {
				continue
			}
			if err := s.LoadRelationships(ctx, r.Scene); err != nil {
				return err
			}
			dupes = append(dupes, s)

			tagIDs = append(tagIDs, missingIDs(append(keep.TagIDs.List(), tagIDs...), s.TagIDs.List())...)
			performerIDs = append(performerIDs, missingIDs(append(keep.PerformerIDs.List(), performerIDs...), s.PerformerIDs.List())...)
			galleryIDs = append(galleryIDs, missingIDs(append(keep.GalleryIDs.List(), galleryIDs...), s.GalleryIDs.List())...)
			for _, g := range s.Groups.List() {
				found := false
				for _, kg := range append(keep.Groups.List(), groups...) {
					if kg.GroupID == g.GroupID {
						found = true
						break
					}
				}
				if !found {
					groups = append(groups, g)
				}
			}
			for _, u := range s.URLs.List() {
				if !slices.Contains(keep.URLs.List(), u) && !slices.Contains(urls, u) {
					urls = append(urls, u)
				}
			}
			for _, sid := range s.StashIDs.List() {
				before := len(stashIDs.StashIDs)
				stashIDs.AddUnique(sid)
				if len(stashIDs.StashIDs) != before {
					stashIDsChanged = true
				}
			}

			// fill in blanks on the scene being kept
			if keep.Title == "" && s.Title != "" && !partial.Title.Set {
				partial.Title = models.NewOptionalString(s.Title)
			}
			if keep.Code == "" && s.Code != "" && !partial.Code.Set {
				partial.Code = models.NewOptionalString(s.Code)
			}
			if keep.Details == "" && s.Details != "" && !partial.Details.Set {
				partial.Details = models.NewOptionalString(s.Details)
			}
			if keep.Director == "" && s.Director != "" && !partial.Director.Set {
				partial.Director = models.NewOptionalString(s.Director)
			}
			if keep.Date == nil && s.Date != nil && !partial.Date.Set {
				partial.Date = models.NewOptionalDate(*s.Date)
			}
			if keep.StudioID == nil && s.StudioID != nil && !partial.StudioID.Set {
				partial.StudioID = models.NewOptionalInt(*s.StudioID)
			}
			if keep.Rating == nil && s.Rating != nil && !partial.Rating.Set {
				partial.Rating = models.NewOptionalInt(*s.Rating)
			}
			if s.Organized {
				organized = true
			}
		}
		if len(dupes) == 0 {
			return nil
		}

		if len(tagIDs) > 0 {
			partial.TagIDs = &models.UpdateIDs{IDs: tagIDs, Mode: models.RelationshipUpdateModeAdd}
		}
		if len(performerIDs) > 0 {
			partial.PerformerIDs = &models.UpdateIDs{IDs: performerIDs, Mode: models.RelationshipUpdateModeAdd}
		}
		if len(galleryIDs) > 0 {
			partial.GalleryIDs = &models.UpdateIDs{IDs: galleryIDs, Mode: models.RelationshipUpdateModeAdd}
		}
		if len(groups) > 0 {
			partial.GroupIDs = &models.UpdateGroupIDs{Groups: groups, Mode: models.RelationshipUpdateModeAdd}
		}
		if len(urls) > 0 {
			partial.URLs = &models.UpdateStrings{Values: urls, Mode: models.RelationshipUpdateModeAdd}
		}
		if stashIDsChanged {
			partial.StashIDs = stashIDs
		}
		if organized && !keep.Organized {
			partial.Organized = models.NewOptionalBool(true)
		}

		if _, err := r.Scene.UpdatePartial(ctx, keep.ID, partial); err != nil {
			return fmt.Errorf("updating scene %d: %w", keep.ID, err)
		}

		for _, s := range dupes {
			if err := sqlite.MergeSceneHistory(ctx, s.ID, keep.ID); err != nil {
				return err
			}
			if err := mgr.SceneService.Destroy(ctx, s, fileDeleter, true, true, false); err != nil {
				return fmt.Errorf("deleting scene %d: %w", s.ID, err)
			}
			logger.Infof("[duplicates] kept %s, removed %s", keep.Path, s.Path)
			removed++
		}
		return nil
	}); err != nil {
		fileDeleter.Rollback()
		return 0, err
	}

	fileDeleter.Commit()
	return removed, nil
}
