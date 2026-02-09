package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"go-mod.ewintr.nl/algorithmic-rss/storage"
)

func main() {
	configPath := flag.String("config", "/home/erik/.config/algorithmicrss/tui.toml", "path to config file")
	flag.Parse()

	conf, err := loadConf(*configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pqCfg := &storage.Config{
		PGHostname: conf["postgres_hostname"],
		PGPort:     conf["postgres_port"],
		PGDBName:   conf["postgres_db_name"],
		PGUser:     conf["postgres_user"],
		PGPassword: conf["postgres_password"],
	}
	pqClient, err := storage.NewClient(pqCfg)
	if err != nil {
		fmt.Printf("could not connect to postgres: %v\n", err)
		os.Exit(1)
	}
	defer pqClient.Close()

	cliRepo := storage.NewCliRepo(pqClient.DB())
	summary := GenerateSummary(cliRepo)
	PrintMatrix(summary)
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
