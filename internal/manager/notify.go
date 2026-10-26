package manager

// Notifications: messages to Discord, Telegram, ntfy, Gotify or any webhook
// when new scenes arrive, a scene cannot be identified, a job fails, or a
// backup or metadata refresh finishes.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
)

// Notification events.
const (
	EventNewScenes = "new_scenes"
	EventUnmatched = "unmatched"
	EventJobFailed = "job_failed"
	EventBackup    = "backup"
	EventResync    = "resync"
)

// NotifyEvents describes every event, in display order.
var NotifyEvents = []struct{ ID, Label string }{
	{EventNewScenes, "New scenes added by auto-import"},
	{EventUnmatched, "Scenes that could not be identified"},
	{EventJobFailed, "A task failed"},
	{EventBackup, "Scheduled backup finished or failed"},
	{EventResync, "Metadata refresh from StashDB finished"},
}

// Notification target kinds.
const (
	NotifyDiscord  = "discord"
	NotifyTelegram = "telegram"
	NotifyNtfy     = "ntfy"
	NotifyGotify   = "gotify"
	NotifyWebhook  = "webhook"
)

// NotifyTarget is somewhere notifications are sent.
type NotifyTarget struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Enabled bool   `json:"enabled"`
	// URL is the Discord webhook, the ntfy topic, the Gotify server or the
	// webhook address. Not used for Telegram.
	URL string `json:"url"`
	// Token is the Telegram bot token, the Gotify app token or an ntfy
	// access token.
	Token string `json:"token"`
	// ChatID is the Telegram chat.
	ChatID string   `json:"chat_id"`
	Events []string `json:"events"`
}

func (t NotifyTarget) wants(event string) bool {
	if !t.Enabled {
		return false
	}
	for _, e := range t.Events {
		if e == event {
			return true
		}
	}
	return false
}

// GetNotifyTargets returns the saved notification targets.
func GetNotifyTargets() []NotifyTarget {
	raw := config.GetInstance().GetNotifyTargetsJSON()
	if raw == "" {
		return []NotifyTarget{}
	}
	var ret []NotifyTarget
	if err := json.Unmarshal([]byte(raw), &ret); err != nil {
		logger.Errorf("[notify] unreadable settings: %v", err)
		return []NotifyTarget{}
	}
	return ret
}

// ValidateNotifyTarget checks a target before it is saved.
func ValidateNotifyTarget(t NotifyTarget) error {
	switch t.Kind {
	case NotifyTelegram:
		if t.Token == "" || t.ChatID == "" {
			return errors.New("enter the Telegram bot token and the chat id")
		}
		return nil
	case NotifyGotify:
		if t.Token == "" {
			return errors.New("enter the Gotify app token")
		}
	case NotifyDiscord, NotifyNtfy, NotifyWebhook:
	default:
		return fmt.Errorf("unknown kind %q", t.Kind)
	}
	u, err := url.Parse(t.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("enter the full address, starting with https://")
	}
	return nil
}

// SetNotifyTargets saves the notification targets.
func SetNotifyTargets(targets []NotifyTarget) error {
	for _, t := range targets {
		if err := ValidateNotifyTarget(t); err != nil {
			name := t.Name
			if name == "" {
				name = t.Kind
			}
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	b, err := json.Marshal(targets)
	if err != nil {
		return err
	}
	cfg := config.GetInstance()
	cfg.SetString(config.NotifyTargets, string(b))
	return cfg.Write()
}

var notifyClient = &http.Client{Timeout: 20 * time.Second}

// throttle stops a burst of failures from flooding a chat
var (
	notifyMu   sync.Mutex
	notifySent = map[string]time.Time{}
)

// Notify sends a message to every target that wants the event. It returns
// straight away; sending happens in the background.
func Notify(event, title, message string) {
	targets := GetNotifyTargets()
	var send []NotifyTarget
	for _, t := range targets {
		if t.wants(event) {
			send = append(send, t)
		}
	}
	if len(send) == 0 {
		return
	}

	key := event + "|" + title
	notifyMu.Lock()
	if last, ok := notifySent[key]; ok && time.Since(last) < time.Minute {
		notifyMu.Unlock()
		return
	}
	notifySent[key] = time.Now()
	notifyMu.Unlock()

	go func() {
		for _, t := range send {
			if err := SendNotification(context.Background(), t, event, title, message); err != nil {
				logger.Warnf("[notify] %s: %v", t.Name, err)
			}
		}
	}()
}

// SendNotification sends one message to one target and waits for the result.
func SendNotification(ctx context.Context, t NotifyTarget, event, title, message string) error {
	var (
		req *http.Request
		err error
	)
	text := title
	if message != "" {
		text = title + "\n" + message
	}
	jsonBody := func(v interface{}) io.Reader {
		b, _ := json.Marshal(v)
		return bytes.NewReader(b)
	}

	switch t.Kind {
	case NotifyDiscord:
		// Discord limits messages to 2000 characters
		if len(text) > 1900 {
			text = text[:1900] + "…"
		}
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, t.URL, jsonBody(map[string]string{
			"username": "Stasharr",
			"content":  "**" + title + "**\n" + message,
		}))
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	case NotifyTelegram:
		if len(text) > 4000 {
			text = text[:4000] + "…"
		}
		api := "https://api.telegram.org/bot" + url.PathEscape(t.Token) + "/sendMessage"
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, api, jsonBody(map[string]interface{}{
			"chat_id":                  t.ChatID,
			"text":                     text,
			"disable_web_page_preview": true,
		}))
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	case NotifyNtfy:
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, t.URL, strings.NewReader(message))
		if req != nil {
			req.Header.Set("Title", title)
			req.Header.Set("Tags", "film_projector")
			if t.Token != "" {
				req.Header.Set("Authorization", "Bearer "+t.Token)
			}
		}
	case NotifyGotify:
		u := strings.TrimRight(t.URL, "/") + "/message?token=" + url.QueryEscape(t.Token)
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, u, jsonBody(map[string]interface{}{
			"title":    title,
			"message":  message,
			"priority": 5,
		}))
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	case NotifyWebhook:
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, t.URL, jsonBody(map[string]interface{}{
			"event":   event,
			"title":   title,
			"message": message,
			"time":    time.Now().Format(time.RFC3339),
		}))
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
			if t.Token != "" {
				req.Header.Set("Authorization", "Bearer "+t.Token)
			}
		}
	default:
		return fmt.Errorf("unknown kind %q", t.Kind)
	}
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Stasharr")

	resp, err := notifyClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%s replied %d: %s", t.Kind, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// listTitles joins up to max titles for a message.
func listTitles(titles []string, max int) string {
	if len(titles) <= max {
		return "• " + strings.Join(titles, "\n• ")
	}
	return "• " + strings.Join(titles[:max], "\n• ") + fmt.Sprintf("\n…and %d more", len(titles)-max)
}

func init() {
	job.OnJobFailed = func(description, err string) {
		Notify(EventJobFailed, "Task failed: "+description, err)
	}
}
