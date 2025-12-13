package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	flag.Parse()

	conf, err := lookupEnv()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mf := NewMiniflux(conf["MINIFLUX_HOSTNAME"], conf["MINIFLUX_API_KEY"])
	pq, err := NewPostgres(
		conf["POSTGRES_HOSTNAME"],
		conf["POSTGRES_PORT"],
		conf["POSTGRES_DB_NAME"],
		conf["POSTGRES_USER"],
		conf["POSTGRES_PASSWORD"],
	)
	if err != nil {
		fmt.Printf("could not open postgres db: %s", err.Error())
		os.Exit(1)
	}
	if err := updatePGCategoriesAndFeeds(mf, pq); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	p := tea.NewProgram(InitialModel(mf, pq), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func lookupEnv() (map[string]string, error) {
	res := make(map[string]string)
	for _, key := range []string{
		"MINIFLUX_HOSTNAME", "MINIFLUX_API_KEY",
		"POSTGRES_HOSTNAME", "POSTGRES_PORT", "POSTGRES_DB_NAME",
		"POSTGRES_USER", "POSTGRES_PASSWORD",
	} {
		val, ok := os.LookupEnv(key)
		if !ok {
			return nil, fmt.Errorf("%s is not set", key)
		}
		res[key] = val
	}

	return res, nil
}

func updatePGCategoriesAndFeeds(mf *Miniflux, pq *Postgres) error {
	mfCats, err := mf.Categories()
	if err != nil {
		return fmt.Errorf("could not fetch miniflux categories: %v", err)
	}
	if err := pq.AddCategories(mfCats); err != nil {
		return fmt.Errorf("could not add postgres categories: %v", err)
	}

	mfFeeds, err := mf.Feeds()
	if err != nil {
		return fmt.Errorf("could not fetch miniflux categories: %v", err)
	}
	if err := pq.AddFeeds(mfFeeds); err != nil {
		return fmt.Errorf("could not add postgres feeds: %v", err)
	}

	return nil
}
