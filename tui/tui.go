package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type CategoriesResult struct {
	Categories []Category
	Error      error
}

type FeedsResult struct {
	Feeds []Feed
	Error error
}

type EntriesResult struct {
	Entries []Entry
	Error   error
}

func (m model) fetchCategories() tea.Cmd {
	return func() tea.Msg {
		cats, err := m.postgres.Categories()
		return CategoriesResult{
			Categories: cats,
			Error:      err,
		}
	}
}

func (m model) fetchFeeds() tea.Cmd {
	return func() tea.Msg {
		feeds, err := m.postgres.Feeds()
		return FeedsResult{
			Feeds: feeds,
			Error: err,
		}
	}
}

func (m model) fetchUnread() tea.Cmd {
	return func() tea.Msg {
		m.status = "Loading..."
		entries, err := m.miniflux.Unread()
		return EntriesResult{Entries: entries, Error: err}
	}
}

type model struct {
	miniflux   *Miniflux
	postgres   *Postgres
	lastUpdate time.Time
	categories map[int64]Category
	feeds      map[int64]Feed
	entries    []Entry
	status     string
	cursor     int
	selected   map[int]struct{}
	quitting   bool
}

func InitialModel(mf *Miniflux, pq *Postgres) model {
	return model{
		miniflux:   mf,
		postgres:   pq,
		categories: make(map[int64]Category, 0),
		feeds:      make(map[int64]Feed, 0),
		entries:    make([]Entry, 0),
		selected:   make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchCategories(),
		m.fetchFeeds(),
		m.fetchUnread(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CategoriesResult:
		if msg.Error != nil {
			m.status = fmt.Sprintf("Error: %s", msg.Error)
			return m, nil
		}
		cats := make(map[int64]Category)
		for _, cat := range msg.Categories {
			cats[cat.ID] = cat
		}
		m.categories = cats
	case FeedsResult:
		if msg.Error != nil {
			m.status = fmt.Sprintf("Error: %s", msg.Error)
			return m, nil
		}
		feeds := make(map[int64]Feed)
		for _, f := range msg.Feeds {
			feeds[f.ID] = f
		}
		m.feeds = feeds

	case EntriesResult:
		if msg.Error != nil {
			m.status = fmt.Sprintf("Error: %s", msg.Error)
			return m, nil
		}
		entries := make([]Entry, 0)
		for _, e := range msg.Entries {
			if m.isVideo(e.FeedID) {
				continue
			}
			entries = append(entries, e)
		}
		m.entries = entries
		// m.status = fmt.Sprintf("Fetched %d entries.", len(m.entries))
		m.lastUpdate = time.Now()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "r":
			return m, m.fetchUnread()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "1", "2", "3":
			if err := m.rateEntry(msg.String()); err != nil {
				m.status = fmt.Sprintf("Error: %s", err)
			}
			m.entries = append(m.entries[:m.cursor], m.entries[m.cursor+1:]...)
			return m, nil
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
	if len(m.entries) > 0 {
		selected := m.entries[m.cursor]
		m.status = fmt.Sprintf("feed: %d, cat: %d", selected.FeedID, m.feeds[selected.FeedID].CategoryID)
	}
	s := fmt.Sprintf("\n Status: %s\n  Last updated: %s)\n\n", m.status, m.lastUpdate.Format("15:04:05"))
	for i := 0; i < len(m.entries) && i < 5; i++ {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, m.entries[i].Title)
	}
	if len(m.entries) > 0 {
		selected := m.entries[m.cursor]
		s += fmt.Sprintf("\n\n%s\n%s\n%s\n\n", m.feeds[selected.FeedID].Title, selected.Title, selected.URL)
		content, err := glamour.Render(selected.Content, "dark")
		if err != nil {
			content = fmt.Sprintf("could not render body: %v", content)
		}
		s += fmt.Sprintf("\n\n%s\n\n", content)
	}
	s += "\nRate: 1: Not opened, 2: Not finished 3: Finished"
	s += "\n\nPress r to refresh, q to quit.\n"

	return s
}

func (m model) rateEntry(rate string) error {
	var rateStr string
	switch rate {
	case "1":
		rateStr = "not_opened"
	case "2":
		rateStr = "not_finished"
	case "3":
		rateStr = "finished"
	default:
		return fmt.Errorf("unknown rating")
	}
	entry := m.entries[m.cursor]
	if err := m.postgres.StoreEntry(entry, rateStr); err != nil {
		return fmt.Errorf("could not store entry: %v", err)
	}
	if err := m.miniflux.MarkRead(entry.ID); err != nil {
		return fmt.Errorf("could not mark entry read: %v", err)
	}

	return nil
}

func (m model) isVideo(feedID int64) bool {
	f, _ := m.feeds[feedID]
	catID := f.CategoryID
	if catID == CAT_VIDEO || catID == CAT_MUSIC {
		return true
	}
	return false
}
