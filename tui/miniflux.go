package main

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"go-mod.ewintr.nl/algorithmic-rss/domain"
	miniflux "miniflux.app/v2/client"
)

type Miniflux struct {
	client *miniflux.Client
}

func NewMiniflux(host, apiKey string) *Miniflux {
	return &Miniflux{
		client: miniflux.NewClient(host, apiKey),
	}
}

func (mf *Miniflux) Categories() ([]domain.Category, error) {
	mfCats, err := mf.client.Categories()
	if err != nil {
		return nil, err
	}
	cats := make([]domain.Category, 0, len(mfCats))
	for _, c := range mfCats {
		cats = append(cats, domain.Category{
			ID:    c.ID,
			Title: c.Title,
		})
	}

	return cats, nil
}

func (mf *Miniflux) Feeds() ([]domain.Feed, error) {
	mfFeeds, err := mf.client.Feeds()
	if err != nil {
		return nil, err
	}
	feeds := make([]domain.Feed, 0, len(mfFeeds))
	for _, f := range mfFeeds {
		feeds = append(feeds, domain.Feed{
			ID:         f.ID,
			CategoryID: f.Category.ID,
			SiteURL:    f.SiteURL,
			FeedURL:    f.FeedURL,
			Title:      f.Title,
		})
	}

	return feeds, nil
}

func (mf *Miniflux) Unread(categoryID int64) ([]domain.Entry, error) {
	entries := make([]domain.Entry, 0)
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
		mfe := domain.Entry{
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
