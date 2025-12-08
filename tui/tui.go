package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	miniflux "miniflux.app/v2/client"
)

type MFResult struct {
	Entries []*miniflux.Entry
	Error   error
}

func (m model) refreshFeed() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Entries(&miniflux.Filter{
			Statuses:  []string{"unread"},
			Order:     "published_at",
			Limit:     5,
			Direction: "desc",
		})

		return MFResult{
			Entries: result.Entries,
			Error:   err,
		}
	}
}

type model struct {
	client     *miniflux.Client
	lastUpdate time.Time
	entries    []string
	cursor     int
	selected   map[int]struct{}
	quitting   bool
}

func InitialModel(client *miniflux.Client) model {
	return model{
		client:   client,
		entries:  make([]string, 0),
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return m.refreshFeed()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MFResult:
		entries := make([]string, 0)
		for _, entry := range msg.Entries {
			entries = append(entries, entry.Title)
		}
		m.entries = entries
		m.lastUpdate = time.Now()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "r":
			return m, m.refreshFeed()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	s := fmt.Sprintf("\n Entries. (Last updated on %s)\n\n", m.lastUpdate.Format("15:04:05"))
	for i, choice := range m.entries {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}
	s += "\nPress r to refresh, q to quit.\n"

	return s
}
