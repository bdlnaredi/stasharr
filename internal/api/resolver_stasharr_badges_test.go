package api

import (
	"reflect"
	"testing"

	"github.com/stashapp/stash/pkg/models"
)

func TestSceneBadges(t *testing.T) {
	g := func(e models.GenderEnum) *models.Performer { return &models.Performer{Gender: &e} }
	tag := func(n string) *models.Tag { return &models.Tag{Name: n} }
	f, m, tf := models.GenderEnumFemale, models.GenderEnumMale, models.GenderEnumTransgenderFemale

	cases := []struct {
		name       string
		performers []*models.Performer
		tags       []*models.Tag
		want       []string
	}{
		{"trans performer", []*models.Performer{g(tf), g(m)}, nil, []string{"trans"}},
		{"trans tag", nil, []*models.Tag{tag("Big Tits"), tag("TS")}, []string{"trans"}},
		{"trans word", nil, []*models.Tag{tag("Trans Anal")}, []string{"trans"}},
		{"not trans", nil, []*models.Tag{tag("Transparent Lingerie"), tag("Tsunami")}, []string{}},
		{"two women", []*models.Performer{g(f), g(f)}, []*models.Tag{tag("Kissing")}, []string{"lesbian"}},
		{"two women and a blowjob", []*models.Performer{g(f), g(f)}, []*models.Tag{tag("Blowjob")}, []string{}},
		{"one woman", []*models.Performer{g(f)}, nil, []string{}},
		{"woman and man", []*models.Performer{g(f), g(m)}, nil, []string{}},
		{"lesbian tag", []*models.Performer{g(f)}, []*models.Tag{tag("Lesbian")}, []string{"lesbian"}},
		{"bisexual", []*models.Performer{g(f), g(m), g(m)}, []*models.Tag{tag("Bisexual MMF")}, []string{"bisexual"}},
		{"bi exact", nil, []*models.Tag{tag("Bi")}, []string{"bisexual"}},
		{"not bi", nil, []*models.Tag{tag("Bikini")}, []string{}},
	}
	for _, c := range cases {
		got := sceneBadges(c.performers, c.tags)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
