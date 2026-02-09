package storage

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

var (
	ErrInvalidConfiguration     = errors.New("invalid configuration")
	ErrPostgresFailure          = errors.New("postgres returned an error")
	ErrNotEnoughSQLMigrations   = errors.New("already more migrations than wanted")
	ErrIncompatibleSQLMigration = errors.New("incompatible migration")
)

type Config struct {
	PGHostname string
	PGPort     string
	PGDBName   string
	PGUser     string
	PGPassword string
}

type Client struct {
	db *sql.DB
}

func NewClient(cfg *Config) (*Client, error) {
	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		cfg.PGHostname, cfg.PGPort, cfg.PGDBName,
		cfg.PGUser, cfg.PGPassword)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
	}

	c := &Client{db: db}

	if err := c.migrate(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) DB() *sql.DB {
	return c.db
}

func (c *Client) migrate() error {
	// Create migration table if not exists
	_, err := c.db.Exec(`
		CREATE TABLE IF NOT EXISTS migration
		(id SERIAL PRIMARY KEY, query TEXT)
	`)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}

	// Find existing migrations
	rows, err := c.db.Query(`SELECT query FROM migration ORDER BY id`)
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
	missing, err := compareMigrations(migrations, existing)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
	}

	for _, query := range missing {
		if _, err := c.db.Exec(query); err != nil {
			return fmt.Errorf("%w: %v", ErrPostgresFailure, err)
		}

		// Register migration
		if _, err := c.db.Exec(`
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
