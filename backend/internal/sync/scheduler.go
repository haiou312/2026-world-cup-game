package sync

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Start launches the background scheduler. One hourly job calls a daily-scoped
// sync: the first run of each day pulls the schedule + results, later runs keep
// refreshing results until every match is final (today_done), then they no-op
// until the next day. Budget control lives in SyncOnce.
func (s *Syncer) Start(ctx context.Context) (*cron.Cron, error) {
	c := cron.New()
	if _, err := c.AddFunc("@hourly", func() {
		if err := s.SyncOnce(ctx, "daily"); err != nil {
			slog.Error("scheduled sync failed", "err", err)
		}
	}); err != nil {
		return nil, err
	}
	c.Start()
	slog.Info("sync scheduler started (@hourly, daily scope)")

	// Kick once on boot so a freshly configured key takes effect promptly.
	go func() {
		if err := s.SyncOnce(ctx, "daily"); err != nil {
			slog.Error("startup sync failed", "err", err)
		}
	}()

	return c, nil
}
