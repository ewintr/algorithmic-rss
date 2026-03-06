package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"go-mod.ewintr.nl/algorithmic-rss/domain"
	miniflux "miniflux.app/v2/client"
)

var (
	TimeoutSmallWeb        = 48 * time.Hour
	TimeoutVids            = 7 * 24 * time.Hour
	TimeoutAggr            = 24 * time.Hour
	KeepEntriesPerCategory = 10
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

	for _, category := range []int64{domain.CatVideo, domain.CatNewsAggregator, domain.CatSmallWeb} {
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

		// Collect all entry IDs
		allEntryIDs := make([]int64, result.Total)
		for i, entry := range result.Entries {
			allEntryIDs[i] = entry.ID
		}

		skipIDs := make([]int64, 0)
		remainingIDs := make([]int64, 0)

		for _, entry := range result.Entries {
			link, err := url.Parse(entry.URL)
			if err != nil {
				catLogger.Error("could not parse url", "url", entry.URL)
				continue
			}
			var shouldSkip bool

			switch category {
			case domain.CatVideo:
				if link.Hostname() == "www.youtube.com" && strings.HasPrefix(link.Path, "/shorts") {
					shouldSkip = true
				}
				if link.Hostname() == "cdn.media.ccc.de" && strings.Contains(link.Path, "-deu-") {
					shouldSkip = true
				}
				if time.Since(entry.Date) > TimeoutVids {
					shouldSkip = true
				}
				// case domain.CatNewsAggregator:
				// 	if time.Since(entry.Date) > TimeoutAggr {
				// 		shouldSkip = true
				// 	}
				// case domain.CatSmallWeb:
				// 	if time.Since(entry.Date) > TimeoutSmallWeb {
				// 		shouldSkip = true
				// }
			}

			if shouldSkip {
				skipIDs = append(skipIDs, entry.ID)
			} else {
				remainingIDs = append(remainingIDs, entry.ID)
			}
		}

		// Pick ten random entries from remainingIDs to keep unread
		keepIDs := make([]int64, 0, KeepEntriesPerCategory)
		for i := 0; i < KeepEntriesPerCategory && len(remainingIDs) > 0; i++ {
			idx := rand.Intn(len(remainingIDs))
			keepIDs = append(keepIDs, remainingIDs[idx])
			remainingIDs = append(remainingIDs[:idx], remainingIDs[idx+1:]...)
		}

		// Mark all rule-matching entries plus remainingIDs as read
		skipIDs = append(skipIDs, remainingIDs...)
		if len(skipIDs) == 0 {
			catLogger.Info("all entries will be kept", "count", len(keepIDs))
			continue
		}
		if err := client.UpdateEntries(skipIDs, "read"); err != nil {
			catLogger.Error("could not mark entries read", "error", err)
			continue
		}

		catLogger.Info("entries processed", "kept", len(keepIDs), "marked_read", len(skipIDs))
	}
}
