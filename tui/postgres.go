package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

var migrations = []string{
	`CREATE TABLE category (
  	  id INTEGER PRIMARY KEY,
  	  title TEXT
	)`,
	`CREATE TABLE feed (
  	id INTEGER PRIMARY KEY,
  	category_id INTEGER references category(id),
  	site_url TEXT,
  	feed_url TEXT,
  	title TEXT
	)`,
	`CREATE TYPE rating AS ENUM(
  	'not_opened', 'not_finished', 'finished'
	)`,
	`CREATE TABLE entry (
  	id INTEGER PRIMARY KEY,
  	feed_id INTEGER references feed(id),
  	updated TIMESTAMP,
  	title TEXT,
  	content TEXT
	)`,
}

var (
	ErrInvalidConfiguration     = errors.New("invalid configuration")
	ErrIncompatibleSQLMigration = errors.New("incompatible migration")
	ErrNotEnoughSQLMigrations   = errors.New("already more migrations than wanted")
	ErrPostgresFailure          = errors.New("postgres returned an error")
)

type Postgres struct {
	db *sql.DB
}

func NewPostgres(host, port, dbname, user, password string) (*Postgres, error) {
	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", host, port, dbname, user, password)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
	}

	p := &Postgres{
		db: db,
	}

	if err := p.migrate(migrations); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Postgres) AddCategories(cats []Category) error {
	values := make([]string, 0)
	for _, c := range cats {
		values = append(values, fmt.Sprintf("(%d, '%s')", c.ID, c.Title))
	}
	query := fmt.Sprintf(`INSERT INTO category
(id, title)
VALUES %s
ON CONFLICT (id)
DO UPDATE SET title = EXCLUDED.title`, strings.Join(values, ","))
	if _, err := p.db.Exec(query); err != nil {
		fmt.Println(query)
		return fmt.Errorf("could not upsert categories: %v", err)
	}
	return nil
}

func (p *Postgres) AddFeeds(feeds []Feed) error {
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
	if _, err := p.db.Exec(query); err != nil {
		fmt.Println(query)
		return fmt.Errorf("could not upsert categories: %v", err)
	}
	return nil
}

func (p *Postgres) Categories() ([]Category, error) {
	rows, err := p.db.Query(`SELECT id, title FROM category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Category, 0)
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Title); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		result = append(result, cat)
	}

	return result, nil
}

func (p *Postgres) Feeds() ([]Feed, error) {
	rows, err := p.db.Query(`SELECT id, category_id, site_url, feed_url, title FROM feed`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Feed, 0)
	for rows.Next() {
		var feed Feed
		if err := rows.Scan(&feed.ID, &feed.CategoryID, &feed.SiteURL, &feed.FeedURL, &feed.Title); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		result = append(result, feed)
	}

	return result, nil
}

func (p *Postgres) migrate(wanted []string) error {
	// Create migration table if not exists
	_, err := p.db.Exec(`
		CREATE TABLE IF NOT EXISTS migration
		(id SERIAL PRIMARY KEY, query TEXT)
	`)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}

	// Find existing migrations
	rows, err := p.db.Query(`SELECT query FROM migration ORDER BY id`)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}
	defer rows.Close()

	var existing []string
	for rows.Next() {
		var query string
		if err := rows.Scan(&query); err != nil {
			return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		existing = append(existing, query)
	}

	// Compare and execute missing migrations
	missing, err := compareMigrations(wanted, existing)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}

	for _, query := range missing {
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}

		// Register migration
		if _, err := p.db.Exec(`
			INSERT INTO migration (query) VALUES ($1)
		`, query); err != nil {
			return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
	}

	return nil
}

func compareMigrations(wanted, existing []string) ([]string, error) {
	var needed []string
	if len(wanted) < len(existing) {
		return nil, ErrNotEnoughSQLMigrations
	}

	for i, want := range wanted {
		switch {
		case i >= len(existing):
			needed = append(needed, want)
		case want == existing[i]:
			// do nothing
		case want != existing[i]:
			return nil, fmt.Errorf("%w: %v", ErrIncompatibleSQLMigration, want)
		}
	}

	return needed, nil
}
