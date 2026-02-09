package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type CliRepo struct {
	db *sql.DB
}

func NewCliRepo(db *sql.DB) *CliRepo {
	return &CliRepo{db: db}
}

func (r *CliRepo) TotalEntries() (int64, error) {
	var count int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM entry").Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}
	return count, nil
}

func (r *CliRepo) EntriesByCategory() (map[int64]int64, error) {
	rows, err := r.db.Query(`
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

func (r *CliRepo) RatingsByStatus() (map[string]int64, error) {
	rows, err := r.db.Query("SELECT rating, COUNT(*) FROM entry GROUP BY rating")
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

func (r *CliRepo) CategoryRatingMatrix() (map[int64]map[string]int64, error) {
	rows, err := r.db.Query(`
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

func (r *CliRepo) AllRatings() ([]string, error) {
	rows, err := r.db.Query("SELECT DISTINCT rating FROM entry ORDER BY rating")
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

func (r *CliRepo) CategoryNames() (map[int64]string, error) {
	rows, err := r.db.Query("SELECT id, title FROM category")
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
