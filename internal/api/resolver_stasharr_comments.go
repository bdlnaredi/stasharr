package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

const maxCommentLength = 1000

func (r *sceneResolver) SiteViews(ctx context.Context, obj *models.Scene) (n int, err error) {
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		n, err = sqlite.SceneViewCount(ctx, obj.ID)
		return err
	})
	return n, err
}

func (r *sceneResolver) SiteCommentCount(ctx context.Context, obj *models.Scene) (n int, err error) {
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		n, err = sqlite.SiteCommentCount(ctx, obj.ID)
		return err
	})
	return n, err
}

func (r *reelResolver) Views(ctx context.Context, obj *models.Reel) (int, error) {
	return obj.Views, nil
}

func (r *reelResolver) CommentCount(ctx context.Context, obj *models.Reel) (n int, err error) {
	err = r.withReadTxn(ctx, func(ctx context.Context) error {
		n, err = sqlite.SiteCommentCount(ctx, obj.SceneID)
		return err
	})
	return n, err
}

func (r *mutationResolver) ReelAddView(ctx context.Context, id string) (int, error) {
	if !access.FromContext(ctx).CanViewReels() {
		return 0, errReelsNotAllowed
	}
	reelID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q", id)
	}
	var n int
	err = r.withTxn(ctx, func(ctx context.Context) error {
		reel, err := r.repository.Reel.Find(ctx, reelID)
		if err != nil {
			return err
		}
		if reel == nil {
			return errors.New("reel not found")
		}
		n, err = sqlite.ReelAddView(ctx, reelID)
		return err
	})
	return n, err
}

// commentAuthor is the name shown with the current user's comments.
func commentAuthor(ctx context.Context) string {
	if p := access.FromContext(ctx); p != nil && p.Username != "" {
		return p.Username
	}
	if u := config.GetInstance().GetUsername(); u != "" {
		return u
	}
	return "Owner"
}

func canDeleteComments(ctx context.Context) bool {
	p := access.FromContext(ctx)
	return p == nil || p.IsOwner() || p.Role == access.RoleAdmin
}

func toSiteComment(ctx context.Context, c *sqlite.SiteComment) *SiteComment {
	mine := c.UserID == sqlite.PlaylistUserID(ctx)
	return &SiteComment{
		ID:        strconv.Itoa(c.ID),
		Author:    c.Author,
		Body:      c.Body,
		CreatedAt: c.CreatedAt.Timestamp,
		FromReel:  c.ReelID != nil,
		Mine:      mine,
		CanDelete: mine || canDeleteComments(ctx),
	}
}

// visibleScene checks the current user may see the scene.
func (r *Resolver) visibleScene(ctx context.Context, sceneID string) (int, error) {
	id, err := strconv.Atoi(sceneID)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q", sceneID)
	}
	visible, err := r.visibleSceneIDs(ctx, []int{id})
	if err != nil {
		return 0, err
	}
	if len(visible) == 0 {
		return 0, errors.New("scene not found")
	}
	return id, nil
}

func (r *queryResolver) SiteComments(ctx context.Context, sceneID string, page *int, perPage *int) (*SiteCommentResult, error) {
	p, pp := 1, 20
	if page != nil && *page > 0 {
		p = *page
	}
	if perPage != nil && *perPage > 0 && *perPage <= 100 {
		pp = *perPage
	}
	ret := &SiteCommentResult{Comments: []*SiteComment{}}
	err := r.withReadTxn(ctx, func(ctx context.Context) error {
		id, err := r.visibleScene(ctx, sceneID)
		if err != nil {
			return err
		}
		list, total, err := sqlite.SiteComments(ctx, id, pp, (p-1)*pp)
		if err != nil {
			return err
		}
		ret.Count = total
		for _, c := range list {
			ret.Comments = append(ret.Comments, toSiteComment(ctx, c))
		}
		return nil
	})
	return ret, err
}

func (r *mutationResolver) SiteAddComment(ctx context.Context, sceneID string, body string, reelID *string) (*SiteComment, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, errors.New("write something first")
	}
	if utf8.RuneCountInString(body) > maxCommentLength {
		return nil, fmt.Errorf("keep comments under %d characters", maxCommentLength)
	}
	var ret *SiteComment
	err := r.withTxn(ctx, func(ctx context.Context) error {
		id, err := r.visibleScene(ctx, sceneID)
		if err != nil {
			return err
		}
		c := &sqlite.SiteComment{
			SceneID: id,
			UserID:  sqlite.PlaylistUserID(ctx),
			Author:  commentAuthor(ctx),
			Body:    body,
		}
		if reelID != nil && *reelID != "" {
			rid, err := strconv.Atoi(*reelID)
			if err != nil {
				return errors.New("invalid reel id")
			}
			c.ReelID = &rid
		}
		if err := sqlite.SiteAddComment(ctx, c); err != nil {
			return err
		}
		ret = toSiteComment(ctx, c)
		return nil
	})
	return ret, err
}

func (r *mutationResolver) SiteDeleteComment(ctx context.Context, id string) (bool, error) {
	cid, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("invalid id %q", id)
	}
	err = r.withTxn(ctx, func(ctx context.Context) error {
		c, err := sqlite.SiteFindComment(ctx, cid)
		if err != nil {
			return err
		}
		if c == nil {
			return errors.New("comment not found")
		}
		if c.UserID != sqlite.PlaylistUserID(ctx) && !canDeleteComments(ctx) {
			return errors.New("you can only delete your own comments")
		}
		return sqlite.SiteDeleteComment(ctx, cid)
	})
	return err == nil, err
}
