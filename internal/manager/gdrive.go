package manager

// Backups to Google Drive. The admin makes a Google OAuth client of the type
// "TVs and Limited Input devices" in their own Google Cloud project, pastes
// its id and secret, and signs in with a code on google.com/device. The
// server keeps the refresh token and uploads each scheduled backup to a
// "Stasharr backups" folder. It only asks for drive.file, so it can only see
// the files it made itself.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/logger"
)

const (
	gdriveScope      = "https://www.googleapis.com/auth/drive.file email"
	gdriveFolderName = "Stasharr backups"
	gdriveDeviceURL  = "https://oauth2.googleapis.com/device/code"
	gdriveTokenURL   = "https://oauth2.googleapis.com/token"
	gdriveFilesURL   = "https://www.googleapis.com/drive/v3/files"
	gdriveUploadURL  = "https://www.googleapis.com/upload/drive/v3/files"
)

var gdriveAPI = &http.Client{Timeout: 60 * time.Second}

// gdriveUploads has no time limit: a backup with images can be large.
var gdriveUploads = &http.Client{}

// ---------------------------------------------------------------- sign in

// GDriveSignIn is what the admin needs to approve the server.
type GDriveSignIn struct {
	UserCode        string
	VerificationURL string
	ExpiresIn       int
}

type gdriveSignInState struct {
	mu      sync.Mutex
	pending bool
	code    *GDriveSignIn
	err     string
	cancel  context.CancelFunc
}

var gdriveState gdriveSignInState

// GDriveSignInStatus returns the sign-in in progress (nil when none) and
// the error of the last one that failed.
func GDriveSignInStatus() (*GDriveSignIn, string) {
	gdriveState.mu.Lock()
	defer gdriveState.mu.Unlock()
	if !gdriveState.pending {
		return nil, gdriveState.err
	}
	return gdriveState.code, ""
}

func gdriveForm(ctx context.Context, endpoint string, form url.Values, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := gdriveAPI.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err := json.Unmarshal(body, out); err != nil {
		return res.StatusCode, fmt.Errorf("google: %s: %s", res.Status, strings.TrimSpace(string(body)))
	}
	return res.StatusCode, nil
}

type gdriveTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

func (t gdriveTokenResponse) problem() string {
	if t.Description != "" {
		return t.Description
	}
	return t.Error
}

// StartGDriveSignIn asks Google for a code and waits in the background for
// the admin to enter it.
func StartGDriveSignIn() (*GDriveSignIn, error) {
	cfg := config.GetInstance()
	id, secret := cfg.GetGDriveClientID(), cfg.GetGDriveClientSecret()
	if id == "" || secret == "" {
		return nil, errors.New("enter the client ID and client secret first")
	}
	var dc struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURL string `json:"verification_url"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
		Description     string `json:"error_description"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := gdriveForm(ctx, gdriveDeviceURL, url.Values{"client_id": {id}, "scope": {gdriveScope}}, &dc); err != nil {
		return nil, err
	}
	if dc.Error != "" || dc.DeviceCode == "" {
		msg := dc.Description
		if msg == "" {
			msg = dc.Error
		}
		if dc.Error == "invalid_client" || dc.Error == "unauthorized_client" {
			msg += " (the client must be of the type \"TVs and Limited Input devices\")"
		}
		return nil, fmt.Errorf("google: %s", msg)
	}
	if dc.Interval < 5 {
		dc.Interval = 5
	}
	code := &GDriveSignIn{UserCode: dc.UserCode, VerificationURL: dc.VerificationURL, ExpiresIn: dc.ExpiresIn}

	gdriveState.mu.Lock()
	if gdriveState.cancel != nil {
		gdriveState.cancel()
	}
	pollCtx, pollCancel := context.WithTimeout(context.Background(), time.Duration(dc.ExpiresIn)*time.Second)
	gdriveState.pending, gdriveState.code, gdriveState.err, gdriveState.cancel = true, code, "", pollCancel
	gdriveState.mu.Unlock()

	go gdrivePoll(pollCtx, id, secret, dc.DeviceCode, time.Duration(dc.Interval)*time.Second)
	return code, nil
}

func gdriveFinish(errMsg string) {
	gdriveState.mu.Lock()
	gdriveState.pending, gdriveState.code, gdriveState.err = false, nil, errMsg
	if gdriveState.cancel != nil {
		gdriveState.cancel()
		gdriveState.cancel = nil
	}
	gdriveState.mu.Unlock()
}

func gdrivePoll(ctx context.Context, id, secret, deviceCode string, every time.Duration) {
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				gdriveFinish("The code expired before it was entered. Try again.")
			}
			return
		case <-time.After(every):
		}
		var t gdriveTokenResponse
		_, err := gdriveForm(ctx, gdriveTokenURL, url.Values{
			"client_id":     {id},
			"client_secret": {secret},
			"device_code":   {deviceCode},
			"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
		}, &t)
		if err != nil {
			if ctx.Err() != nil {
				continue
			}
			logger.Warnf("[gdrive] sign in: %v", err)
			continue
		}
		switch t.Error {
		case "":
		case "authorization_pending":
			continue
		case "slow_down":
			every += 5 * time.Second
			continue
		case "access_denied":
			gdriveFinish("Access was denied on the Google page.")
			return
		default:
			gdriveFinish("Google: " + t.problem())
			return
		}
		if t.RefreshToken == "" {
			gdriveFinish("Google sent no refresh token. Remove Stasharr's access at myaccount.google.com/permissions and try again.")
			return
		}
		cfg := config.GetInstance()
		cfg.SetString(config.GDriveRefreshToken, t.RefreshToken)
		cfg.SetString(config.GDriveEmail, emailFromIDToken(t.IDToken))
		cfg.SetString(config.GDriveFolderID, "")
		cfg.SetString(config.GDriveLastError, "")
		cfg.SetBool(config.GDriveEnabled, true)
		if err := cfg.Write(); err != nil {
			logger.Warnf("[gdrive] saving sign in: %v", err)
		}
		gdriveCache.forget()
		logger.Infof("[gdrive] connected to %s", cfg.GetGDriveEmail())
		gdriveFinish("")
		return
	}
}

// CancelGDriveSignIn stops waiting for a code.
func CancelGDriveSignIn() {
	gdriveFinish("")
}

// DisconnectGDrive forgets the Google account. Files on Drive stay.
func DisconnectGDrive() error {
	cfg := config.GetInstance()
	if tok := cfg.GetGDriveRefreshToken(); tok != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var ignore map[string]any
		_, _ = gdriveForm(ctx, "https://oauth2.googleapis.com/revoke", url.Values{"token": {tok}}, &ignore)
	}
	cfg.SetString(config.GDriveRefreshToken, "")
	cfg.SetString(config.GDriveEmail, "")
	cfg.SetString(config.GDriveFolderID, "")
	cfg.SetBool(config.GDriveEnabled, false)
	gdriveCache.forget()
	return cfg.Write()
}

// emailFromIDToken reads the email from the token Google sent over TLS.
func emailFromIDToken(tok string) string {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		return ""
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
	}
	_ = json.Unmarshal(b, &claims)
	return claims.Email
}

// ---------------------------------------------------------------- access tokens

type gdriveTokenCache struct {
	mu      sync.Mutex
	token   string
	expires time.Time
}

var gdriveCache gdriveTokenCache

func (c *gdriveTokenCache) forget() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

func gdriveAccessToken(ctx context.Context) (string, error) {
	gdriveCache.mu.Lock()
	defer gdriveCache.mu.Unlock()
	if gdriveCache.token != "" && time.Now().Before(gdriveCache.expires) {
		return gdriveCache.token, nil
	}
	cfg := config.GetInstance()
	if cfg.GetGDriveRefreshToken() == "" {
		return "", errors.New("google drive is not connected")
	}
	var t gdriveTokenResponse
	if _, err := gdriveForm(ctx, gdriveTokenURL, url.Values{
		"client_id":     {cfg.GetGDriveClientID()},
		"client_secret": {cfg.GetGDriveClientSecret()},
		"refresh_token": {cfg.GetGDriveRefreshToken()},
		"grant_type":    {"refresh_token"},
	}, &t); err != nil {
		return "", err
	}
	if t.Error != "" {
		if t.Error == "invalid_grant" {
			return "", errors.New("google no longer accepts the sign in: connect Google Drive again, and if this happens every week set the app's publishing status to In production in Google Cloud")
		}
		return "", errors.New("google: " + t.problem())
	}
	gdriveCache.token = t.AccessToken
	gdriveCache.expires = time.Now().Add(time.Duration(t.ExpiresIn-60) * time.Second)
	return t.AccessToken, nil
}

func gdriveDo(ctx context.Context, client *http.Client, method, endpoint string, body io.Reader, size int64, contentType string, out any) (*http.Response, error) {
	tok, err := gdriveAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if size >= 0 {
		req.ContentLength = size
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		defer res.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(b, &e)
		msg := e.Error.Message
		if msg == "" {
			msg = strings.TrimSpace(string(b))
		}
		if strings.Contains(msg, "has not been used in project") || strings.Contains(msg, "is disabled") {
			msg = "turn on the Google Drive API in your Google Cloud project: " + msg
		}
		return nil, fmt.Errorf("google drive: %s: %s", res.Status, msg)
	}
	if out != nil {
		defer res.Body.Close()
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ---------------------------------------------------------------- files

// GDriveFile is a backup on Google Drive.
type GDriveFile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size,string"`
	CreatedAt time.Time `json:"createdTime"`
}

func gdriveFolder(ctx context.Context) (string, error) {
	cfg := config.GetInstance()
	if id := cfg.GetGDriveFolderID(); id != "" {
		var f struct {
			ID      string `json:"id"`
			Trashed bool   `json:"trashed"`
		}
		if _, err := gdriveDo(ctx, gdriveAPI, http.MethodGet, gdriveFilesURL+"/"+url.PathEscape(id)+"?fields=id,trashed", nil, -1, "", &f); err == nil && !f.Trashed {
			return id, nil
		}
	}
	// find one this app made before, or make it
	q := url.Values{
		"q":      {fmt.Sprintf("name = '%s' and mimeType = 'application/vnd.google-apps.folder' and trashed = false", gdriveFolderName)},
		"fields": {"files(id)"},
	}
	var list struct {
		Files []struct {
			ID string `json:"id"`
		} `json:"files"`
	}
	if _, err := gdriveDo(ctx, gdriveAPI, http.MethodGet, gdriveFilesURL+"?"+q.Encode(), nil, -1, "", &list); err != nil {
		return "", err
	}
	var id string
	if len(list.Files) > 0 {
		id = list.Files[0].ID
	} else {
		meta, err := json.Marshal(map[string]any{"name": gdriveFolderName, "mimeType": "application/vnd.google-apps.folder"})
		if err != nil {
			return "", err
		}
		var f struct {
			ID string `json:"id"`
		}
		if _, err := gdriveDo(ctx, gdriveAPI, http.MethodPost, gdriveFilesURL+"?fields=id", strings.NewReader(string(meta)), int64(len(meta)), "application/json", &f); err != nil {
			return "", err
		}
		id = f.ID
	}
	cfg.SetString(config.GDriveFolderID, id)
	_ = cfg.Write()
	return id, nil
}

// ListGDriveBackups lists the backups in the Drive folder, newest first.
func ListGDriveBackups(ctx context.Context) ([]GDriveFile, error) {
	folder, err := gdriveFolder(ctx)
	if err != nil {
		return nil, err
	}
	var all []GDriveFile
	page := ""
	for {
		q := url.Values{
			"q":        {fmt.Sprintf("'%s' in parents and trashed = false", folder)},
			"orderBy":  {"createdTime desc"},
			"fields":   {"nextPageToken, files(id, name, size, createdTime)"},
			"pageSize": {"200"},
		}
		if page != "" {
			q.Set("pageToken", page)
		}
		var list struct {
			Next  string       `json:"nextPageToken"`
			Files []GDriveFile `json:"files"`
		}
		if _, err := gdriveDo(ctx, gdriveAPI, http.MethodGet, gdriveFilesURL+"?"+q.Encode(), nil, -1, "", &list); err != nil {
			return nil, err
		}
		all = append(all, list.Files...)
		if list.Next == "" {
			break
		}
		page = list.Next
	}
	return all, nil
}

// UploadToGDrive uploads a backup file and deletes the oldest backups on
// Drive beyond the number to keep.
func UploadToGDrive(ctx context.Context, path string) error {
	err := uploadToGDrive(ctx, path)
	cfg := config.GetInstance()
	if err != nil {
		cfg.SetString(config.GDriveLastError, err.Error())
		_ = cfg.Write()
		Notify(EventBackup, "Upload to Google Drive failed", err.Error())
		return err
	}
	cfg.SetString(config.GDriveLastError, "")
	cfg.SetLastRun(config.GDriveLastUpload, time.Now())
	_ = cfg.Write()
	return nil
}

func uploadToGDrive(ctx context.Context, path string) error {
	folder, err := gdriveFolder(ctx)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	name := filepath.Base(path)
	meta, err := json.Marshal(map[string]any{"name": name, "parents": []string{folder}})
	if err != nil {
		return err
	}
	res, err := gdriveDo(ctx, gdriveAPI, http.MethodPost, gdriveUploadURL+"?uploadType=resumable", strings.NewReader(string(meta)), int64(len(meta)), "application/json; charset=UTF-8", nil)
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	session := res.Header.Get("Location")
	if session == "" {
		return errors.New("google drive: no upload address")
	}
	started := time.Now()
	up, err := gdriveDo(ctx, gdriveUploads, http.MethodPut, session, f, info.Size(), "application/octet-stream", nil)
	if err != nil {
		return err
	}
	_ = up.Body.Close()
	logger.Infof("[gdrive] uploaded %s (%s) in %s", name, humanSize(info.Size()), time.Since(started).Round(time.Second))

	// keep the newest ones
	list, err := ListGDriveBackups(ctx)
	if err != nil {
		return nil
	}
	keep := config.GetInstance().GetGDriveKeep()
	for i, b := range list {
		if i < keep {
			continue
		}
		res, err := gdriveDo(ctx, gdriveAPI, http.MethodDelete, gdriveFilesURL+"/"+url.PathEscape(b.ID), nil, -1, "", nil)
		if err != nil {
			logger.Warnf("[gdrive] deleting %s: %v", b.Name, err)
			continue
		}
		_ = res.Body.Close()
	}
	return nil
}

// DownloadFromGDrive saves a Drive backup in the backup folder, where it
// shows in the list of backups and can be restored.
func DownloadFromGDrive(ctx context.Context, fileID string) (string, error) {
	var meta GDriveFile
	if _, err := gdriveDo(ctx, gdriveAPI, http.MethodGet, gdriveFilesURL+"/"+url.PathEscape(fileID)+"?fields=id,name,size,createdTime", nil, -1, "", &meta); err != nil {
		return "", err
	}
	name := filepath.Base(meta.Name)
	if _, ok := backupSchemaVersion(filepath.Base(GetInstance().Config.GetDatabasePath()), name); !ok {
		return "", fmt.Errorf("%s is not a Stasharr backup", name)
	}
	dir := GetInstance().Config.GetBackupDirectoryPathOrDefault()
	dest := filepath.Join(dir, name)
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	res, err := gdriveDo(ctx, gdriveUploads, http.MethodGet, gdriveFilesURL+"/"+url.PathEscape(fileID)+"?alt=media", nil, -1, "", nil)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, res.Body); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	_ = os.Chtimes(dest, meta.CreatedAt, meta.CreatedAt)
	return dest, nil
}

// gdriveReady says whether scheduled backups should go to Drive.
func gdriveReady() bool {
	cfg := config.GetInstance()
	return cfg.GetGDriveEnabled() && cfg.GetGDriveRefreshToken() != "" && cfg.GetGDriveClientID() != ""
}
