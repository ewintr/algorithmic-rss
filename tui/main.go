package main

import (
	"fmt"
	"os"

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

	p := tea.NewProgram(InitialModel(client))
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
