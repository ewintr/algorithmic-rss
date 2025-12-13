package main

import (
	"fmt"

	miniflux "miniflux.app/v2/client"
)

const (
	CAT_PERSONAL   = int64(3)
	CAT_COMPANY    = int64(4)
	CAT_AGGREGATOR = int64(6)
	CAT_MUSIC      = int64(8)
	CAT_VIDEO      = int64(7)
)

type Category struct {
	ID    int64
	Title string
}

type Feed struct {
	ID         int64
	CategoryID int64
	FeedURL    string
	SiteURL    string
	Title      string
}

type Entry struct {
	ID      int64
	FeedID  int64
	Title   string
	URL     string
	Content string
}

type Miniflux struct {
	client *miniflux.Client
}

func NewMiniflux(host, apiKey string) *Miniflux {
	return &Miniflux{
		client: miniflux.NewClient(host, apiKey),
	}
}

func (mf *Miniflux) Categories() ([]Category, error) {
	mfCats, err := mf.client.Categories()
	if err != nil {
		return nil, err
	}
	cats := make([]Category, 0, len(mfCats))
	for _, c := range mfCats {
		cats = append(cats, Category{
			ID:    c.ID,
			Title: c.Title,
		})
	}

	return cats, nil
}

func (mf *Miniflux) Feeds() ([]Feed, error) {
	mfFeeds, err := mf.client.Feeds()
	if err != nil {
		return nil, err
	}
	feeds := make([]Feed, 0, len(mfFeeds))
	for _, f := range mfFeeds {
		feeds = append(feeds, Feed{
			ID:         f.ID,
			CategoryID: f.Category.ID,
			SiteURL:    f.SiteURL,
			FeedURL:    f.FeedURL,
			Title:      f.Title,
		})
	}

	return feeds, nil
}

func (mf *Miniflux) Unread() ([]Entry, error) {
	entries := make([]Entry, 0)
	result, err := mf.client.Entries(&miniflux.Filter{
		Statuses:  []string{"unread"},
		Order:     "published_at",
		Direction: "asc",
	})
	if err != nil {
		return nil, err
	}
	for _, e := range result.Entries {
		if e.Feed == nil {
			return nil, fmt.Errorf("could not fetch unread entries, entry without feed: %d", e.ID)
		}
		mfe := Entry{
			ID:      e.ID,
			FeedID:  e.Feed.ID,
			Title:   e.Title,
			URL:     e.URL,
			Content: e.Content,
		}
		entries = append(entries, mfe)
	}

	return entries, nil
}
