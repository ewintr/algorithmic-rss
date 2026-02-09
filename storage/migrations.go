package storage

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
  	rating rating,
  	title TEXT,
  	url TEXT,
  	content TEXT
	)`,
	`ALTER TYPE rating ADD VALUE 'only_comments'`,
}
