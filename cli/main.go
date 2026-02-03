package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

func main() {
	configPath := flag.String("config", "/home/erik/.config/algorithmicrss/tui.toml", "path to config file")
	flag.Parse()

	conf, err := loadConf(*configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pq, err := NewPostgresFromConfig(conf)
	if err != nil {
		fmt.Printf("could not connect to postgres: %v\n", err)
		os.Exit(1)
	}

	summary := GenerateSummary(pq)
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
