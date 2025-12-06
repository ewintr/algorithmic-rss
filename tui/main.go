package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	miniflux "miniflux.app/v2/client"
)

func main() {
	mfHostname, ok := os.LookupEnv("MINIFLUX_HOSTNAME")
	if !ok {
		fmt.Println("MINIFLUX_HOSTNAME not set")
		os.Exit(1)
	}
	mfApiKey, ok := os.LookupEnv("MINIFLUX_API_KEY")
	if !ok {
		fmt.Println("MINIFLUX_API_KEY not set")
		os.Exit(1)
	}
	client := miniflux.NewClient(mfHostname, mfApiKey)
	feed := make(chan MFResult)
	done := make(chan bool)
	go MFListener(client, feed, done)

	p := tea.NewProgram(InitialModel(feed, done))
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type MFResult struct {
	Entries []*miniflux.Entry
	Error   error
}

func MFListener(client *miniflux.Client, feed chan MFResult, done chan bool) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			result, err := client.Entries(&miniflux.Filter{
				Statuses:  []string{"unread"},
				Order:     "published_at",
				Direction: "desc",
			})
			feed <- MFResult{
				Entries: result.Entries,
				Error:   err,
			}
		case <-done:
			return
		}
	}
}
