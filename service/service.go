package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
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
	logger.Info("checking for things to mark as read")
	skipIDs := make([]int64, 0)
	for _, entry := range result.Entries {
		link, err := url.Parse(entry.URL)
		if err != nil {
			logger.Error("could not parse url", "url", entry.URL)
			continue
		}
		if link.Hostname() == "www.youtube.com" && strings.HasPrefix(link.Path, "/shorts") {
			skipIDs = append(skipIDs, entry.ID)
		}
		if link.Hostname() == "cdn.media.ccc.de" && strings.Contains(link.Path, "-deu-") {
			skipIDs = append(skipIDs, entry.ID)
		}
		if time.Since(entry.Date) < 3*24*time.Hour {
			skipIDs = append(skipIDs, entry.ID)
		}
	}
	if len(skipIDs) == 0 {
		logger.Info("nothing to skip")
		return nil
	}
	if err := client.UpdateEntries(skipIDs, "read"); err != nil {
		return fmt.Errorf("could not mark entries read: %v", err)
	}
	logger.Info("youtube shorts marked read", "count", len(skipIDs))

	return nil
}
