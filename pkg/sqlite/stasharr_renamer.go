package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/stashapp/stash/pkg/models"
)

// RenameCaptions updates the caption file names of a video file after the
// video was renamed from oldStem to newStem (file names without extension).
func RenameCaptions(ctx context.Context, fileID models.FileID, oldStem, newStem string) error {
	type captionRow struct {
		Lang     string `db:"language_code"`
		Filename string `db:"filename"`
		Type     string `db:"caption_type"`
	}
	var rows []captionRow
	if err := dbWrapper.Select(ctx, &rows,
		"SELECT language_code, filename, caption_type FROM video_captions WHERE file_id = ?", int(fileID)); err != nil {
		return fmt.Errorf("reading captions: %w", err)
	}
	for _, r := range rows {
		if !strings.HasPrefix(r.Filename, oldStem) {
			continue
		}
		name := newStem + strings.TrimPrefix(r.Filename, oldStem)
		if _, err := dbWrapper.Exec(ctx,
			"UPDATE video_captions SET filename = ? WHERE file_id = ? AND language_code = ? AND caption_type = ?",
			name, int(fileID), r.Lang, r.Type); err != nil {
			return fmt.Errorf("renaming caption: %w", err)
		}
	}
	return nil
}
