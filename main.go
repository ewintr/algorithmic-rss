package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	miniflux "miniflux.app/v2/client"
)

const (
	CAT_VIDEO = int64(2)
)

func main() {
	hostname, ok := os.LookupEnv("MINIFLUX_HOSTNAME")
	if !ok {
		fmt.Println("MINIFLUX_HOSTNAME not set")
		os.Exit(1)
	}
	apiKey, ok := os.LookupEnv("MINIFLUX_API_KEY")
	if !ok {
		fmt.Println("MINIFLUX_API_KEY not set")
		os.Exit(1)
	}

	ctx := context.Background()
	client := miniflux.NewClient(hostname, apiKey)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("starting service")

	ticker := time.NewTicker(10 * time.Minute)
	c := make(chan os.Signal, 1)
	for {
		select {
		case <-ticker.C:
			if err := checkUnread(ctx, client, logger); err != nil {
				logger.Error("error checking feed", "error", err)
			}
		case <-c:
			logger.Info("stopping service")
			goto EXIT
		}
	}

EXIT:
	ticker.Stop()
	logger.Info("service exited")
}

func checkUnread(ctx context.Context, client *miniflux.Client, logger *slog.Logger) error {
	logger.Info("checking feed...")

	result, err := client.CategoryEntriesContext(ctx, CAT_VIDEO, &miniflux.Filter{Statuses: []string{"unread"}})
	if err != nil {
		return fmt.Errorf("could not fetch entries: %v", err)
	}
	if result.Total == 0 {
		logger.Info("no unread entries found")
		return nil
	}

	logger.Info("unread video entries found", "count", result.Total)
	logger.Info("checking for youtube shorts")
	ytShortIDs := make([]int64, 0)
	for _, entry := range result.Entries {
		if strings.HasPrefix(entry.URL, "https://www.youtube.com/shorts") {
			ytShortIDs = append(ytShortIDs, entry.ID)
		}
	}
	if len(ytShortIDs) == 0 {
		logger.Info("no shorts found")
		return nil
	}
	if err := client.UpdateEntries(ytShortIDs, "read"); err != nil {
		return fmt.Errorf("could not mark entries read: %v", err)
	}
	logger.Info("youtube shorts marked read", "count", len(ytShortIDs))

	return nil
}
