package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	tea "github.com/charmbracelet/bubbletea"
	"go-mod.ewintr.nl/algorithmic-rss/storage"
)

var (
	configPath = flag.String("config", "/home/erik/.config/algorithmicrss/tui.toml", "path to config file")
)

func main() {
	flag.Parse()

	conf, err := loadConf(*configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mf := NewMiniflux(conf["miniflux_hostname"], conf["miniflux_api_key"])

	pqCfg := &storage.Config{
		PGHostname: conf["postgres_hostname"],
		PGPort:     conf["postgres_port"],
		PGDBName:   conf["postgres_db_name"],
		PGUser:     conf["postgres_user"],
		PGPassword: conf["postgres_password"],
	}
	pqClient, err := storage.NewClient(pqCfg)
	if err != nil {
		fmt.Printf("could not open postgres db: %s", err.Error())
		os.Exit(1)
	}
	defer pqClient.Close()

	tuiRepo := storage.NewTuiRepo(pqClient.DB())
	if err := updatePGCategoriesAndFeeds(mf, tuiRepo); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	p := tea.NewProgram(InitialModel(mf, tuiRepo), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadConf(path string) (map[string]string, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	var config map[string]string
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}

	return config, nil
}

func updatePGCategoriesAndFeeds(mf *Miniflux, repo *storage.TuiRepo) error {
	mfCats, err := mf.Categories()
	if err != nil {
		return fmt.Errorf("could not fetch miniflux categories: %v", err)
	}
	if err := repo.AddCategories(mfCats); err != nil {
		return fmt.Errorf("could not add postgres categories: %v", err)
	}

	mfFeeds, err := mf.Feeds()
	if err != nil {
		return fmt.Errorf("could not fetch miniflux categories: %v", err)
	}
	if err := repo.AddFeeds(mfFeeds); err != nil {
		return fmt.Errorf("could not add postgres feeds: %v", err)
	}

	return nil
}
