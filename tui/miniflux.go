package main

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	miniflux "miniflux.app/v2/client"
)

const (
	CAT_PERSONAL   = int64(3)
	CAT_COMPANY    = int64(4)
	CAT_AGGREGATOR = int64(6)
	CAT_MUSIC      = int64(8)
	CAT_VIDEO      = int64(2)
	CAT_PROJECT    = int64(5)
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

func (mf *Miniflux) Unread(categoryID int64) ([]Entry, error) {
	entries := make([]Entry, 0)
	result, err := mf.client.Entries(&miniflux.Filter{
		Statuses:   []string{"unread"},
		CategoryID: categoryID,
		Order:      "published_at",
		Direction:  "asc",
	})
	if err != nil {
		return nil, err
	}
	for _, e := range result.Entries {
		if e.Feed == nil {
			return nil, fmt.Errorf("could not fetch unread entries, entry without feed: %d", e.ID)
		}
		mdContent := ConvertHTMLToMarkdown(e.Content)
		// if len(mdContent) > 500 {
		// 	mdContent = mdContent[:500]
		// }
		mfe := Entry{
			ID:      e.ID,
			FeedID:  e.Feed.ID,
			Title:   e.Title,
			URL:     e.URL,
			Content: mdContent,
		}
		entries = append(entries, mfe)
	}

	return entries, nil
}

func (mf *Miniflux) MarkRead(id ...int64) error {
	if err := mf.client.UpdateEntries(id, "read"); err != nil {
		return fmt.Errorf("could not mark entries read: %v", err)
	}

	return nil
}

func ConvertHTMLToMarkdown(html string) string {
	markdown, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		markdown = fmt.Sprintf("Error: could not convert html: %v", err)
	}

	return markdown
}
