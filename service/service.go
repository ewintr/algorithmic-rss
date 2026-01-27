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
	CatVideo         = int64(2)
	CatNewsAggrators = int64(6)
)

var (
	TimeoutVids = 48 * time.Hour
	TimeoutAggr = 24 * time.Hour
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
			checkUnread(ctx, client, logger)
		case <-c:
			logger.Info("stopping service")
			goto EXIT
		}
	}

EXIT:
	ticker.Stop()
	logger.Info("service exited")
}

func checkUnread(ctx context.Context, client *miniflux.Client, logger *slog.Logger) {
	logger.Info("checking feed...")

	for _, category := range []int64{CatVideo, CatNewsAggrators} {
		catLogger := logger.With("category", category)
		result, err := client.CategoryEntriesContext(ctx, category, &miniflux.Filter{Statuses: []string{"unread"}})
		if err != nil {
			catLogger.Error("could not fetch entries", "error", err)
			continue
		}
		if result.Total == 0 {
			catLogger.Info("no unread entries found")
			continue
		}

		catLogger.Info("unread entries found", "count", result.Total)
		catLogger.Info("checking for things to mark as read")
		skipIDs := make([]int64, 0)
		for _, entry := range result.Entries {
			link, err := url.Parse(entry.URL)
			if err != nil {
				catLogger.Error("could not parse url", "url", entry.URL)
				continue
			}
			switch category {
			case CatVideo:
				if link.Hostname() == "www.youtube.com" && strings.HasPrefix(link.Path, "/shorts") {
					skipIDs = append(skipIDs, entry.ID)
				}
				if link.Hostname() == "cdn.media.ccc.de" && strings.Contains(link.Path, "-deu-") {
					skipIDs = append(skipIDs, entry.ID)
				}
				if time.Since(entry.Date) > TimeoutVids {
					skipIDs = append(skipIDs, entry.ID)
				}
			case CatNewsAggrators:
				if time.Since(entry.Date) > TimeoutAggr {
					skipIDs = append(skipIDs, entry.ID)
				}
			}
		}
		if len(skipIDs) == 0 {
			catLogger.Info("nothing to skip")
			continue
		}
		if err := client.UpdateEntries(skipIDs, "read"); err != nil {
			catLogger.Error("could not mark entries read", "error", err)
			continue
		}
		catLogger.Info("entries marked read", "count", len(skipIDs))
	}
}
