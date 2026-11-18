package manager

// Stasharr: Netflix-style profiles. A profile is a user account of its own,
// so its history, likes and My list stay apart, and it belongs to a main
// account. Switching profile moves the browser session to that account.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
)

// MaxProfiles is how many profiles a main account can have besides itself.
const MaxProfiles = 5

var (
	ErrNotMainProfile = errors.New("only the main profile can change profiles")
	ErrWrongPin       = errors.New("wrong PIN")
	ErrPinLocked      = errors.New("too many wrong PINs; try again in 15 minutes")
	pinPattern        = regexp.MustCompile(`^[0-9]{4,8}$`)
	colorPattern      = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// ProfileInfo is one profile of a family, the main account included.
type ProfileInfo struct {
	ID      int // the account; 0 is the owner
	Name    string
	Color   string
	HasPin  bool
	Main    bool
	Current bool
}

func familyMainName(ctx context.Context, r models.Repository, mainID int) (string, error) {
	if mainID == 0 {
		return GetInstance().Config.GetUsername(), nil
	}
	u, err := r.User.Find(ctx, mainID)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", ErrUserNotFound
	}
	return u.Username, nil
}

// Profiles lists the signed-in account's family: the main account first,
// then its profiles.
func Profiles(ctx context.Context, p *access.Principal) ([]ProfileInfo, error) {
	if p == nil {
		return nil, nil
	}
	r := GetInstance().Repository
	main := p.FamilyID()
	var ret []ProfileInfo
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		name, err := familyMainName(ctx, r, main)
		if err != nil {
			return err
		}
		pin, err := sqlite.FindPin(ctx, main)
		if err != nil {
			return err
		}
		ret = append(ret, ProfileInfo{ID: main, Name: name, HasPin: pin != "", Main: true, Current: !p.Profile})
		rows, err := sqlite.ProfilesOf(ctx, main)
		if err != nil {
			return err
		}
		for _, row := range rows {
			pin, err := sqlite.FindPin(ctx, row.UserID)
			if err != nil {
				return err
			}
			ret = append(ret, ProfileInfo{
				ID: row.UserID, Name: row.Name, Color: row.Color, HasPin: pin != "",
				Current: p.Profile && p.UserID == row.UserID,
			})
		}
		return nil
	})
	return ret, err
}

func cleanProfileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("give the profile a name")
	}
	if len([]rune(name)) > 32 {
		return "", errors.New("the name is too long")
	}
	return name, nil
}

func cleanColor(c string) string {
	if colorPattern.MatchString(c) {
		return c
	}
	return ""
}

// inFamily finds a profile of the signed-in account's family.
func inFamily(ctx context.Context, p *access.Principal, id int) (*sqlite.Profile, error) {
	row, err := sqlite.FindProfile(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil || row.ParentID != p.FamilyID() {
		return nil, errors.New("that profile is not one of yours")
	}
	return row, nil
}

// CreateProfile adds a profile to the signed-in main account.
func CreateProfile(ctx context.Context, p *access.Principal, name, color string) (*ProfileInfo, error) {
	if p == nil {
		return nil, errors.New("sign in first")
	}
	if p.Profile {
		return nil, ErrNotMainProfile
	}
	name, err := cleanProfileName(name)
	if err != nil {
		return nil, err
	}
	r := GetInstance().Repository
	var info *ProfileInfo
	err = r.WithTxn(ctx, func(ctx context.Context) error {
		rows, err := sqlite.ProfilesOf(ctx, p.UserID)
		if err != nil {
			return err
		}
		if len(rows) >= MaxProfiles {
			return fmt.Errorf("an account can have up to %d profiles", MaxProfiles)
		}
		for _, row := range rows {
			if strings.EqualFold(row.Name, name) {
				return fmt.Errorf("you already have a profile called %q", name)
			}
		}
		b := make([]byte, 4)
		_, _ = rand.Read(b)
		u := &models.User{
			// never used to sign in: profiles are reached by switching
			Username:     fmt.Sprintf("profile-%d-%s", p.UserID, hex.EncodeToString(b)),
			PasswordHash: "!",
			Role:         access.RoleViewer,
		}
		if err := r.User.Create(ctx, u); err != nil {
			return err
		}
		row := sqlite.Profile{UserID: u.ID, ParentID: p.UserID, Name: name, Color: cleanColor(color)}
		if err := sqlite.CreateProfile(ctx, row); err != nil {
			return err
		}
		info = &ProfileInfo{ID: u.ID, Name: row.Name, Color: row.Color}
		return nil
	})
	if err != nil {
		return nil, err
	}
	GetInstance().UserService.Invalidate()
	return info, nil
}

// UpdateProfile renames a profile or changes its colour.
func UpdateProfile(ctx context.Context, p *access.Principal, id int, name, color string) error {
	if p == nil || p.Profile {
		return ErrNotMainProfile
	}
	name, err := cleanProfileName(name)
	if err != nil {
		return err
	}
	r := GetInstance().Repository
	err = r.WithTxn(ctx, func(ctx context.Context) error {
		row, err := inFamily(ctx, p, id)
		if err != nil {
			return err
		}
		row.Name, row.Color = name, cleanColor(color)
		return sqlite.UpdateProfile(ctx, *row)
	})
	if err == nil {
		GetInstance().UserService.Invalidate()
	}
	return err
}

// DeleteProfile removes a profile with its history, likes and lists.
func DeleteProfile(ctx context.Context, p *access.Principal, id int) error {
	if p == nil || p.Profile {
		return ErrNotMainProfile
	}
	r := GetInstance().Repository
	err := r.WithTxn(ctx, func(ctx context.Context) error {
		if _, err := inFamily(ctx, p, id); err != nil {
			return err
		}
		if err := sqlite.SetPin(ctx, id, ""); err != nil {
			return err
		}
		if err := sqlite.ForgetAccountData(ctx, id); err != nil {
			return err
		}
		return r.User.Destroy(ctx, id)
	})
	if err == nil {
		GetInstance().UserService.Invalidate()
	}
	return err
}

// DeleteProfilesOf removes every profile of a main account, when that
// account is deleted.
func DeleteProfilesOf(ctx context.Context, mainID int) error {
	r := GetInstance().Repository
	rows, err := sqlite.ProfilesOf(ctx, mainID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := sqlite.SetPin(ctx, row.UserID, ""); err != nil {
			return err
		}
		if err := sqlite.ForgetAccountData(ctx, row.UserID); err != nil {
			return err
		}
		if err := r.User.Destroy(ctx, row.UserID); err != nil {
			return err
		}
	}
	return sqlite.SetPin(ctx, mainID, "")
}

// SetProfilePin sets or (with "") removes the PIN of the main profile or
// one of the profiles.
func SetProfilePin(ctx context.Context, p *access.Principal, id int, pin string) error {
	if p == nil || p.Profile {
		return ErrNotMainProfile
	}
	hash := ""
	if pin != "" {
		if !pinPattern.MatchString(pin) {
			return errors.New("a PIN is 4 to 8 digits")
		}
		b, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		hash = string(b)
	}
	r := GetInstance().Repository
	return r.WithTxn(ctx, func(ctx context.Context) error {
		if id != p.UserID {
			if _, err := inFamily(ctx, p, id); err != nil {
				return err
			}
		}
		return sqlite.SetPin(ctx, id, hash)
	})
}

// pinGuard stops PIN guessing: 5 wrong PINs lock a profile for 15 minutes.
type pinGuard struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

var profilePins = &pinGuard{fails: map[string][]time.Time{}}

func (g *pinGuard) recent(key string) []time.Time {
	cut := time.Now().Add(-15 * time.Minute)
	var keep []time.Time
	for _, t := range g.fails[key] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	g.fails[key] = keep
	return keep
}

// SwitchProfile checks the PIN and returns the account name the session
// should move to.
func SwitchProfile(ctx context.Context, p *access.Principal, id int, pin string) (string, error) {
	if p == nil {
		return "", errors.New("sign in first")
	}
	r := GetInstance().Repository
	key := fmt.Sprintf("%d:%d", p.FamilyID(), id)

	profilePins.mu.Lock()
	locked := len(profilePins.recent(key)) >= 5
	profilePins.mu.Unlock()
	if locked {
		return "", ErrPinLocked
	}

	var target string
	err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		main := p.FamilyID()
		if id != main {
			if _, err := inFamily(ctx, p, id); err != nil {
				return err
			}
			u, err := r.User.Find(ctx, id)
			if err != nil {
				return err
			}
			if u == nil {
				return ErrUserNotFound
			}
			target = u.Username
		} else {
			name, err := familyMainName(ctx, r, main)
			if err != nil {
				return err
			}
			target = name
		}
		// moving to the profile you are on needs no PIN
		if id == p.UserID && p.Profile == (id != main) {
			return nil
		}
		hash, err := sqlite.FindPin(ctx, id)
		if err != nil {
			return err
		}
		if hash != "" && bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)) != nil {
			return ErrWrongPin
		}
		return nil
	})
	if errors.Is(err, ErrWrongPin) {
		profilePins.mu.Lock()
		profilePins.fails[key] = append(profilePins.recent(key), time.Now())
		profilePins.mu.Unlock()
	}
	if err != nil {
		return "", err
	}
	return target, nil
}
