package manager

// Reels picked by the Stasharr Tagger: it looks at a frame every few
// seconds and scores how much is happening (the AI marker labels, such as
// Blowjob or Cowgirl) and how good the picture is. Talking, title cards,
// dark or blurry shots and empty rooms score low. Each reel of a scene
// then shows a different act where it can.

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

// ReelSourceAI marks reels whose moments the tagger chose.
const ReelSourceAI = "AI"

// extra descriptions for how good a frame looks in a reel
var reelLookPrompts = []struct {
	prompt string
	weight float64
}{
	{"a sharp, well lit, close photo of two naked people having sex | an erotic, high quality photo of a naked woman", 0.6},
	{"a dark, blurry or out of focus photo | a black screen", -1.0},
	{"text, a logo or a title on a plain background | a website advertisement", -1.2},
}

// aiReelScores returns a score for each second of the video and, for each
// second, the act seen there (an index into acts, or -1). It returns nil
// when the tagger is not set up or not reachable, so the other ways of
// picking moments are used.
func aiReelScores(ctx context.Context, vf *models.VideoFile) (scores []float64, act []int, acts []string) {
	cfg := GetInstance().Config
	url := cfg.GetAIURL()
	if url == "" || !cfg.GetReelsAI() {
		return nil, nil, nil
	}
	st, err := GetTaggerStatus(ctx, url)
	if err != nil || !st.OK {
		return nil, nil, nil
	}
	ensemble := st.Has("ensemble")

	var labels []AILabel
	for _, l := range AILabels(cfg) {
		if l.Enabled {
			labels = append(labels, l)
		}
	}
	if len(labels) == 0 {
		return nil, nil, nil
	}
	prompt := func(p string) string {
		if ensemble {
			return p
		}
		return strings.TrimSpace(strings.Split(p, "|")[0])
	}
	prompts := make([]string, 0, len(labels)+len(reelLookPrompts))
	for _, l := range labels {
		prompts = append(prompts, prompt(l.Prompt))
	}
	for _, p := range reelLookPrompts {
		prompts = append(prompts, prompt(p.prompt))
	}

	total := vf.Duration
	interval := math.Max(4, total/300)
	frames, err := sampleFrames(ctx, vf.Path, interval)
	if err != nil || len(frames) == 0 {
		logger.Warnf("[reels] tagger frames for %s: %v", vf.Path, err)
		return nil, nil, nil
	}
	var probs [][]float64
	for i := 0; i < len(frames); i += 32 {
		end := min(i+32, len(frames))
		res, err := classify(ctx, url, prompts, frames[i:end], false)
		if err != nil {
			logger.Warnf("[reels] tagger: %v", err)
			return nil, nil, nil
		}
		probs = append(probs, res.Scores...)
	}

	// labels kept for the start or end of a scene (a cumshot) count only
	// there, as for markers
	length := float64(len(probs)) * interval
	for _, l := range labels {
		if l.Marker {
			acts = append(acts, l.Name)
		}
	}
	frameScore := make([]float64, len(probs))
	frameAct := make([]int, len(probs))
	for f, row := range probs {
		at := float64(f) * interval
		action, other := 0.0, 0.0
		best, bestP := -1, 0.0
		a := 0
		for l, lab := range labels {
			if l >= len(row) {
				break
			}
			p := row[l]
			if (lab.When == "end" && at < length*(1-AILabelPartOfScene)) || (lab.When == "start" && at > length*AILabelPartOfScene) {
				p = 0
			}
			if lab.Marker {
				action += p
				if p > bestP {
					best, bestP = a, p
				}
				a++
			} else {
				other += p
			}
		}
		look := 0.0
		for k, lp := range reelLookPrompts {
			if i := len(labels) + k; i < len(row) {
				look += lp.weight * row[i]
			}
		}
		frameScore[f] = action - 0.8*other + look
		if bestP >= 0.25 {
			frameAct[f] = best
		} else {
			frameAct[f] = -1
		}
	}

	n := int(total) + 1
	scores = make([]float64, n)
	act = make([]int, n)
	for s := 0; s < n; s++ {
		f := int(float64(s) / interval)
		if f >= len(frameScore) {
			f = len(frameScore) - 1
		}
		scores[s] = frameScore[f]
		act[s] = frameAct[f]
	}
	return zscores(scores, true), act, acts
}

// pickVariedPeaks picks n windows like pickPeaks, but after each pick the
// moments showing the same act count for less, so a scene's reels show
// different acts where the scene has them.
func pickVariedPeaks(scores []float64, act []int, n int, segLen, total float64) []models.ReelSegment {
	w := int(math.Ceil(segLen))
	s := append([]float64(nil), scores...)
	var out []models.ReelSegment
	for k := 0; k < n; k++ {
		picks := pickPeaks(s, 1, segLen, total)
		if len(picks) == 0 {
			break
		}
		p := picks[0]
		start := int(p.Start)
		// a window over an earlier pick means nothing good is left
		blocked := false
		for i := start; i < start+w && i < len(s); i++ {
			if s[i] <= -1e5 {
				blocked = true
				break
			}
		}
		if blocked {
			break
		}
		out = append(out, p)

		// the act this reel shows
		counts := map[int]int{}
		for i := start; i < start+w && i < len(act); i++ {
			if act[i] >= 0 {
				counts[act[i]]++
			}
		}
		shown, most := -1, 0
		for a, c := range counts {
			if c > most {
				shown, most = a, c
			}
		}
		// keep the next reels away from this one, and from the same act
		for i := max(0, start-w); i < start+w+w/2 && i < len(s); i++ {
			s[i] = -1e6
		}
		if shown >= 0 {
			for i := range s {
				if i < len(act) && act[i] == shown && s[i] > -1e5 {
					s[i] -= 0.8
				}
			}
		}
	}
	if len(out) < n {
		// not enough different moments: fill up the usual way
		rest := pickPeaks(scores, n, segLen, total)
		for _, r := range rest {
			if len(out) >= n {
				break
			}
			clash := false
			for _, o := range out {
				if r.Start < o.End+float64(w)/2 && o.Start < r.End+float64(w)/2 {
					clash = true
					break
				}
			}
			if !clash {
				out = append(out, r)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}
