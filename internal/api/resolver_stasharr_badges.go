package api

// Ribbons on scene cards and pages: Trans, Lesbian or Bisexual, worked out
// from the performers' genders and the scene's tags.

import (
	"context"
	"strings"
	"unicode"

	"github.com/stashapp/stash/internal/api/loaders"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/models"
)

var (
	badgeTransWords   = []string{"trans", "transsexual", "transsexuals", "transgender", "tgirl", "tgirls", "shemale", "shemales", "ladyboy", "ladyboys", "tranny", "travesti"}
	badgeTransNames   = []string{"ts", "t-girl", "t girl", "trans woman", "trans man"}
	badgeLesbianWords = []string{"lesbian", "lesbians", "lesbo", "sapphic"}
	badgeLesbianNames = []string{"girl on girl", "girl/girl", "girl-girl", "gg", "girls only", "all girl", "all girls"}
	badgeBiWords      = []string{"bisexual", "bisexuals", "bisex"}
	badgeBiNames      = []string{"bi", "bi mmf", "mmf bi", "bi-sexual", "bi sexual"}
	badgeNotGirlsOnly = []string{"blowjob", "blowjobs", "bj", "cumshot", "cumshots", "facial", "facials", "creampie", "handjob", "titjob", "deepthroat", "ffm", "mmf", "fmm", "threesome", "foursome", "gangbang", "orgy", "boy", "boys", "guy", "guys", "man", "men", "male", "penis", "cock", "dick", "straight", "bbc"}
)

func badgeWords(name string) []string {
	return strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

func badgeMatch(name string, words, names []string) bool {
	n := strings.TrimSpace(strings.ToLower(name))
	for _, x := range names {
		if n == x {
			return true
		}
	}
	for _, w := range badgeWords(n) {
		for _, x := range words {
			if w == x {
				return true
			}
		}
	}
	return false
}

// sceneBadges works out a scene's ribbons.
func sceneBadges(performers []*models.Performer, tags []*models.Tag) []string {
	trans, lesbian, bi := false, false, false
	notGirlsOnly := false
	for _, t := range tags {
		if t == nil {
			continue
		}
		names := []string{t.Name}
		if t.Aliases.Loaded() {
			names = append(names, t.Aliases.List()...)
		}
		for _, name := range names {
			trans = trans || badgeMatch(name, badgeTransWords, badgeTransNames)
			lesbian = lesbian || badgeMatch(name, badgeLesbianWords, badgeLesbianNames)
			bi = bi || badgeMatch(name, badgeBiWords, badgeBiNames)
			notGirlsOnly = notGirlsOnly || badgeMatch(name, badgeNotGirlsOnly, nil)
		}
	}
	women, others := 0, 0
	for _, p := range performers {
		if p == nil || p.Gender == nil {
			others++
			continue
		}
		switch *p.Gender {
		case models.GenderEnumTransgenderFemale, models.GenderEnumTransgenderMale:
			trans = true
			others++
		case models.GenderEnumFemale:
			women++
		default:
			others++
		}
	}
	// two or more women and nobody else, unless a tag says a man is there
	// too (performer lists often leave the men out)
	if women >= 2 && others == 0 && !notGirlsOnly && !trans && !bi {
		lesbian = true
	}
	ret := []string{}
	if trans {
		ret = append(ret, "trans")
	}
	if lesbian && !trans {
		ret = append(ret, "lesbian")
	}
	if bi {
		ret = append(ret, "bisexual")
	}
	return ret
}

func (r *sceneResolver) StasharrBadges(ctx context.Context, obj *models.Scene) ([]string, error) {
	if !config.GetInstance().GetSceneBadgesEnabled() {
		return []string{}, nil
	}
	if !obj.PerformerIDs.Loaded() || !obj.TagIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			if err := obj.LoadPerformerIDs(ctx, r.repository.Scene); err != nil {
				return err
			}
			return obj.LoadTagIDs(ctx, r.repository.Scene)
		}); err != nil {
			return nil, err
		}
	}
	performers, _ := loaders.From(ctx).PerformerByID.LoadAll(obj.PerformerIDs.List())
	tags, _ := loaders.From(ctx).TagByID.LoadAll(obj.TagIDs.List())
	return sceneBadges(performers, tags), nil
}
