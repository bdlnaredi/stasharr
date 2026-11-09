package api

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/job"
)

func googleDriveBackup() *GoogleDriveBackup {
	cfg := config.GetInstance()
	ret := &GoogleDriveBackup{
		ClientID:        cfg.GetGDriveClientID(),
		HasClientSecret: cfg.GetGDriveClientSecret() != "",
		Connected:       cfg.GetGDriveRefreshToken() != "",
		Enabled:         cfg.GetGDriveEnabled(),
		Keep:            cfg.GetGDriveKeep(),
		FolderName:      "Stasharr backups",
		LastUpload:      lastRun(cfg, config.GDriveLastUpload),
	}
	if e := cfg.GetGDriveEmail(); e != "" {
		ret.Email = &e
	}
	if e := cfg.GetGDriveLastError(); e != "" {
		ret.LastError = &e
	}
	code, signInErr := manager.GDriveSignInStatus()
	if code != nil {
		ret.SignIn = &GoogleDriveSignIn{UserCode: code.UserCode, VerificationURL: code.VerificationURL, ExpiresIn: code.ExpiresIn}
	}
	if signInErr != "" {
		ret.SignInError = &signInErr
	}
	return ret
}

func (r *queryResolver) GoogleDriveBackup(ctx context.Context) (*GoogleDriveBackup, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return googleDriveBackup(), nil
}

func (r *queryResolver) GoogleDriveFiles(ctx context.Context) ([]*GoogleDriveFile, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	ret := []*GoogleDriveFile{}
	if config.GetInstance().GetGDriveRefreshToken() == "" {
		return ret, nil
	}
	list, err := manager.ListGDriveBackups(ctx)
	if err != nil {
		return nil, err
	}
	for _, f := range list {
		ret = append(ret, &GoogleDriveFile{ID: f.ID, Name: f.Name, Size: int(f.Size), CreatedAt: f.CreatedAt})
	}
	return ret, nil
}

func (r *mutationResolver) ConfigureGoogleDrive(ctx context.Context, input GoogleDriveBackupInput) (*GoogleDriveBackup, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	cfg := config.GetInstance()
	if input.ClientID != nil {
		id := strings.TrimSpace(*input.ClientID)
		if id != cfg.GetGDriveClientID() && cfg.GetGDriveRefreshToken() != "" {
			// a different client cannot use the old sign in
			if err := manager.DisconnectGDrive(); err != nil {
				return nil, err
			}
		}
		cfg.SetString(config.GDriveClientID, id)
	}
	if input.ClientSecret != nil && strings.TrimSpace(*input.ClientSecret) != "" {
		cfg.SetString(config.GDriveClientSecret, strings.TrimSpace(*input.ClientSecret))
	}
	if input.Enabled != nil {
		if *input.Enabled && cfg.GetGDriveRefreshToken() == "" {
			return nil, errors.New("connect a Google account first")
		}
		cfg.SetBool(config.GDriveEnabled, *input.Enabled)
	}
	if input.Keep != nil {
		k := *input.Keep
		if k < 1 {
			k = 1
		}
		if k > 365 {
			k = 365
		}
		cfg.SetInt(config.GDriveKeep, k)
	}
	if err := cfg.Write(); err != nil {
		return nil, err
	}
	return googleDriveBackup(), nil
}

func (r *mutationResolver) StartGoogleDriveSignIn(ctx context.Context) (*GoogleDriveSignIn, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	code, err := manager.StartGDriveSignIn()
	if err != nil {
		return nil, err
	}
	return &GoogleDriveSignIn{UserCode: code.UserCode, VerificationURL: code.VerificationURL, ExpiresIn: code.ExpiresIn}, nil
}

func (r *mutationResolver) CancelGoogleDriveSignIn(ctx context.Context) (bool, error) {
	if err := requireAdmin(ctx); err != nil {
		return false, err
	}
	manager.CancelGDriveSignIn()
	return true, nil
}

func (r *mutationResolver) DisconnectGoogleDrive(ctx context.Context) (bool, error) {
	if err := requireAdmin(ctx); err != nil {
		return false, err
	}
	if err := manager.DisconnectGDrive(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) UploadBackupToGoogleDrive(ctx context.Context, id string) (string, error) {
	if err := requireAdmin(ctx); err != nil {
		return "", err
	}
	mgr := manager.GetInstance()
	path, err := mgr.BackupFilePath(id)
	if err != nil {
		return "", err
	}
	if config.GetInstance().GetGDriveRefreshToken() == "" {
		return "", errors.New("connect a Google account first")
	}
	jobID := mgr.JobManager.Add(context.Background(), "Upload backup to Google Drive", job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		return manager.UploadToGDrive(ctx, path)
	}))
	return strconv.Itoa(jobID), nil
}

func (r *mutationResolver) DownloadGoogleDriveBackup(ctx context.Context, id string) (string, error) {
	if err := requireAdmin(ctx); err != nil {
		return "", err
	}
	mgr := manager.GetInstance()
	jobID := mgr.JobManager.Add(context.Background(), "Copy backup from Google Drive", job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		_, err := manager.DownloadFromGDrive(ctx, id)
		return err
	}))
	return strconv.Itoa(jobID), nil
}
