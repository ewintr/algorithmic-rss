package storage

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"go-mod.ewintr.nl/algorithmic-rss/domain"
)

type TuiRepo struct {
	db *sql.DB
}

func NewTuiRepo(db *sql.DB) *TuiRepo {
	return &TuiRepo{db: db}
}

func (r *TuiRepo) Categories() ([]domain.Category, error) {
	rows, err := r.db.Query(`SELECT id, title FROM category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.Category, 0)
	for rows.Next() {
		var cat domain.Category
		if err := rows.Scan(&cat.ID, &cat.Title); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		result = append(result, cat)
	}

	return result, nil
}

func (r *TuiRepo) AddCategories(cats []domain.Category) error {
	values := make([]string, 0)
	for _, c := range cats {
		values = append(values, fmt.Sprintf("(%d, '%s')", c.ID, c.Title))
	}
	query := fmt.Sprintf(`INSERT INTO category
(id, title)
VALUES %s
ON CONFLICT (id)
DO UPDATE SET title = EXCLUDED.title`, strings.Join(values, ","))
	if _, err := r.db.Exec(query); err != nil {
		fmt.Println(query)
		return fmt.Errorf("could not upsert categories: %v", err)
	}
	return nil
}

func (r *TuiRepo) Feeds() ([]domain.Feed, error) {
	rows, err := r.db.Query(`SELECT id, category_id, site_url, feed_url, title FROM feed`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.Feed, 0)
	for rows.Next() {
		var feed domain.Feed
		if err := rows.Scan(&feed.ID, &feed.CategoryID, &feed.SiteURL, &feed.FeedURL, &feed.Title); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		result = append(result, feed)
	}

	return result, nil
}

func (r *TuiRepo) AddFeeds(feeds []domain.Feed) error {
	values := make([]string, 0)
	for _, f := range feeds {
		values = append(values, fmt.Sprintf("(%d, %d, %s, %s, %s)",
			f.ID, f.CategoryID, pq.QuoteLiteral(f.FeedURL),
			pq.QuoteLiteral(f.SiteURL), pq.QuoteLiteral(f.Title)))
	}
	query := fmt.Sprintf(`INSERT INTO feed
(id, category_id, feed_url, site_url, title)
VALUES %s
ON CONFLICT (id)
DO UPDATE SET
category_id = EXCLUDED.category_id,
feed_url = EXCLUDED.feed_url,
site_url = EXCLUDED.site_url,
title = EXCLUDED.title`, strings.Join(values, ","))
	if _, err := r.db.Exec(query); err != nil {
		fmt.Println(query)
		return fmt.Errorf("could not upsert categories: %v", err)
	}
	return nil
}

func (r *TuiRepo) StoreEntry(entry domain.Entry, rating string) error {
	if _, err := r.db.Exec(`INSERT INTO entry
(id, feed_id, updated, title, rating, url, content)
VALUES ($1, $2, NOW(), $3, $4, $5, $6)`,
		entry.ID, entry.FeedID, entry.Title,
		rating, entry.URL, entry.Content,
	); err != nil {
		return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}

	return nil
}
