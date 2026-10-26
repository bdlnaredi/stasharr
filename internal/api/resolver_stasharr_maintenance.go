package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/access"
	"github.com/stashapp/stash/pkg/job"
)

func requireAdmin(ctx context.Context) error {
	return access.RequireRole(ctx, access.RoleAdmin)
}

func toNotifyTarget(t manager.NotifyTarget) *NotifyTarget {
	events := t.Events
	if events == nil {
		events = []string{}
	}
	return &NotifyTarget{
		ID:      t.ID,
		Name:    t.Name,
		Kind:    t.Kind,
		Enabled: t.Enabled,
		URL:     t.URL,
		Token:   t.Token,
		ChatID:  t.ChatID,
		Events:  events,
	}
}

func fromNotifyInput(in NotifyTargetInput) manager.NotifyTarget {
	t := manager.NotifyTarget{
		Name:    in.Name,
		Kind:    in.Kind,
		Enabled: in.Enabled,
		Events:  in.Events,
	}
	if in.ID != nil {
		t.ID = *in.ID
	}
	if in.URL != nil {
		t.URL = *in.URL
	}
	if in.Token != nil {
		t.Token = *in.Token
	}
	if in.ChatID != nil {
		t.ChatID = *in.ChatID
	}
	if t.Name == "" {
		t.Name = t.Kind
	}
	return t
}

func newTargetID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (r *queryResolver) NotifyTargets(ctx context.Context) ([]*NotifyTarget, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	ret := []*NotifyTarget{}
	for _, t := range manager.GetNotifyTargets() {
		ret = append(ret, toNotifyTarget(t))
	}
	return ret, nil
}

func (r *queryResolver) NotifyEvents(ctx context.Context) ([]*NotifyEvent, error) {
	ret := []*NotifyEvent{}
	for _, e := range manager.NotifyEvents {
		ret = append(ret, &NotifyEvent{ID: e.ID, Label: e.Label})
	}
	return ret, nil
}

func (r *mutationResolver) SaveNotifyTargets(ctx context.Context, targets []*NotifyTargetInput) ([]*NotifyTarget, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	list := make([]manager.NotifyTarget, 0, len(targets))
	for _, in := range targets {
		if in == nil {
			continue
		}
		t := fromNotifyInput(*in)
		if t.ID == "" {
			t.ID = newTargetID()
		}
		list = append(list, t)
	}
	if err := manager.SetNotifyTargets(list); err != nil {
		return nil, err
	}
	return r.Query().NotifyTargets(ctx)
}

func (r *mutationResolver) TestNotifyTarget(ctx context.Context, target NotifyTargetInput) (bool, error) {
	if err := requireAdmin(ctx); err != nil {
		return false, err
	}
	t := fromNotifyInput(target)
	if err := manager.ValidateNotifyTarget(t); err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if err := manager.SendNotification(ctx, t, "test", "Stasharr test message", "Notifications are working."); err != nil {
		return false, err
	}
	return true, nil
}

func lastRun(cfg *config.Config, key string) *time.Time {
	t := cfg.GetLastRun(key)
	if t.IsZero() {
		return nil
	}
	return &t
}

func maintenanceConfig() *MaintenanceConfig {
	cfg := config.GetInstance()
	boxes := []string{}
	for _, b := range cfg.GetStashBoxes() {
		name := b.Name
		if name == "" {
			name = b.Endpoint
		}
		boxes = append(boxes, name)
	}
	return &MaintenanceConfig{
		Hour:               cfg.GetMaintenanceHour(),
		BackupEnabled:      cfg.GetBackupScheduleEnabled(),
		BackupKeep:         cfg.GetBackupKeep(),
		BackupIncludeBlobs: cfg.GetBackupIncludeBlobs(),
		BackupDir:          cfg.GetBackupDirectoryPathOrDefault(),
		BackupLastRun:      lastRun(cfg, config.BackupLastRun),
		ResyncEnabled:      cfg.GetResyncEnabled(),
		ResyncIntervalDays: cfg.GetResyncIntervalDays(),
		ResyncScenes:       cfg.GetResyncScenes(),
		ResyncPerformers:   cfg.GetResyncPerformers(),
		ResyncStudios:      cfg.GetResyncStudios(),
		ResyncLastRun:      lastRun(cfg, config.ResyncLastRun),
		StashBoxes:         boxes,
	}
}

func (r *queryResolver) MaintenanceConfig(ctx context.Context) (*MaintenanceConfig, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return maintenanceConfig(), nil
}

func (r *mutationResolver) ConfigureMaintenance(ctx context.Context, input MaintenanceConfigInput) (*MaintenanceConfig, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	cfg := config.GetInstance()
	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}
	if input.Hour != nil {
		cfg.SetInt(config.MaintenanceHour, clamp(*input.Hour, 0, 23))
	}
	if input.BackupEnabled != nil {
		cfg.SetBool(config.BackupScheduleEnabled, *input.BackupEnabled)
	}
	if input.BackupKeep != nil {
		cfg.SetInt(config.BackupKeep, clamp(*input.BackupKeep, 1, 365))
	}
	if input.BackupIncludeBlobs != nil {
		cfg.SetBool(config.BackupIncludeBlobs, *input.BackupIncludeBlobs)
	}
	if input.ResyncEnabled != nil {
		cfg.SetBool(config.ResyncEnabled, *input.ResyncEnabled)
	}
	if input.ResyncIntervalDays != nil {
		cfg.SetInt(config.ResyncIntervalDays, clamp(*input.ResyncIntervalDays, 1, 365))
	}
	if input.ResyncScenes != nil {
		cfg.SetBool(config.ResyncScenes, *input.ResyncScenes)
	}
	if input.ResyncPerformers != nil {
		cfg.SetBool(config.ResyncPerformers, *input.ResyncPerformers)
	}
	if input.ResyncStudios != nil {
		cfg.SetBool(config.ResyncStudios, *input.ResyncStudios)
	}
	if err := cfg.Write(); err != nil {
		return nil, err
	}
	return maintenanceConfig(), nil
}

func (r *queryResolver) DatabaseBackups(ctx context.Context) ([]*DatabaseBackup, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	list, err := manager.GetInstance().ListBackups()
	if err != nil {
		return nil, err
	}
	ret := []*DatabaseBackup{}
	for _, b := range list {
		ret = append(ret, &DatabaseBackup{
			ID:            b.ID,
			Name:          b.Name,
			Scheduled:     b.Scheduled,
			Size:          int(b.Size),
			CreatedAt:     b.CreatedAt,
			SchemaVersion: b.SchemaVersion,
			IncludesBlobs: b.IncludesBlobs,
		})
	}
	return ret, nil
}

func (r *mutationResolver) RunScheduledBackup(ctx context.Context) (string, error) {
	if err := requireAdmin(ctx); err != nil {
		return "", err
	}
	mgr := manager.GetInstance()
	id := mgr.JobManager.Add(context.Background(), "Backup", job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		_, err := mgr.ScheduledBackup()
		return err
	}))
	return strconv.Itoa(id), nil
}

func (r *mutationResolver) RestoreDatabaseBackup(ctx context.Context, id string) (bool, error) {
	if err := requireAdmin(ctx); err != nil {
		return false, err
	}
	if err := manager.GetInstance().RestoreBackup(context.Background(), id); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) DeleteDatabaseBackup(ctx context.Context, id string) (bool, error) {
	if err := requireAdmin(ctx); err != nil {
		return false, err
	}
	if err := manager.GetInstance().DeleteBackup(id); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) RunMetadataResync(ctx context.Context) (string, error) {
	if err := requireAdmin(ctx); err != nil {
		return "", err
	}
	mgr := manager.GetInstance()
	mgr.Config.SetLastRun(config.ResyncLastRun, time.Now())
	_ = mgr.Config.Write()
	return strconv.Itoa(mgr.StartResync(context.Background())), nil
}
