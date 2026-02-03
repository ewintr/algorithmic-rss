package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

var (
	ErrInvalidConfiguration     = errors.New("invalid configuration")
	ErrPostgresFailure          = errors.New("postgres returned an error")
)

type Postgres struct {
	db *sql.DB
}

func NewPostgresFromConfig(config map[string]string) (*Postgres, error) {
	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		config["postgres_hostname"], config["postgres_port"],
		config["postgres_db_name"], config["postgres_user"], config["postgres_password"])
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
	}

	return &Postgres{db: db}, nil
}

func (p *Postgres) TotalEntries() (int64, error) {
	var count int64
	if err := p.db.QueryRow("SELECT COUNT(*) FROM entry").Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}
	return count, nil
}

func (p *Postgres) EntriesByCategory() (map[int64]int64, error) {
	rows, err := p.db.Query(`
		SELECT feed.category_id, COUNT(*)
		FROM entry
		JOIN feed ON entry.feed_id = feed.id
		GROUP BY feed.category_id
	`)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}
	defer rows.Close()

	result := make(map[int64]int64)
	for rows.Next() {
		var categoryID int64
		var count int64
		if err := rows.Scan(&categoryID, &count); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		result[categoryID] = count
	}

	return result, nil
}

func (p *Postgres) RatingsByStatus() (map[string]int64, error) {
	rows, err := p.db.Query("SELECT rating, COUNT(*) FROM entry GROUP BY rating")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var rating string
		var count int64
		if err := rows.Scan(&rating, &count); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		result[rating] = count
	}

	return result, nil
}

func (p *Postgres) CategoryRatingMatrix() (map[int64]map[string]int64, error) {
	rows, err := p.db.Query(`
		SELECT feed.category_id, entry.rating, COUNT(*)
		FROM entry
		JOIN feed ON entry.feed_id = feed.id
		GROUP BY feed.category_id, entry.rating
	`)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}
	defer rows.Close()

	result := make(map[int64]map[string]int64)
	for rows.Next() {
		var categoryID int64
		var rating string
		var count int64
		if err := rows.Scan(&categoryID, &rating, &count); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}

		if result[categoryID] == nil {
			result[categoryID] = make(map[string]int64)
		}
		result[categoryID][rating] = count
	}

	return result, nil
}

func (p *Postgres) AllRatings() ([]string, error) {
	rows, err := p.db.Query("SELECT DISTINCT rating FROM entry ORDER BY rating")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}
	defer rows.Close()

	var ratings []string
	for rows.Next() {
		var rating string
		if err := rows.Scan(&rating); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		ratings = append(ratings, rating)
	}

	return ratings, nil
}

func (p *Postgres) CategoryNames() (map[int64]string, error) {
	rows, err := p.db.Query("SELECT id, title FROM category")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}
		result[id] = title
	}

	return result, nil
}
