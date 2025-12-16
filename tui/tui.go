package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
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

type MarkReadResult error

func (m model) rateEntry(entry Entry, rate string) tea.Cmd {
	return func() tea.Msg {
		var rateStr string
		switch rate {
		case "1":
			rateStr = "not_opened"
		case "2":
			rateStr = "not_finished"
		case "3":
			rateStr = "finished"
		default:
			return MarkReadResult(fmt.Errorf("unknown rating"))
		}
		if err := m.postgres.StoreEntry(entry, rateStr); err != nil {
			return MarkReadResult(fmt.Errorf("could not store entry: %v", err))
		}
		if err := m.miniflux.MarkRead(entry.ID); err != nil {
			return MarkReadResult(fmt.Errorf("could not mark entry read: %v", err))
		}

		return MarkReadResult(nil)
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
	width      int
	height     int
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
	case MarkReadResult:
		if msg != nil {
			m.status = fmt.Sprintf("Error: %s", error(msg))
		}
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
			if m.cursor < 4 && m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "1", "2", "3":
			entry := m.entries[m.cursor]
			m.entries = append(m.entries[:m.cursor], m.entries[m.cursor+1:]...)
			return m, m.rateEntry(entry, msg.String())
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
	if m.width == 0 {
		return "loading..."
	}
	list := lipgloss.NewStyle().
		Width(m.width).
		Height(9).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Padding(1, 2).
		Render(m.listView())

	help := lipgloss.NewStyle().
		Width(m.width).
		Height(2).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Padding(1, 2).
		Render(m.helpView())

	entry := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-lipgloss.Height(list)-lipgloss.Height(help)).
		MaxHeight(m.height-lipgloss.Height(list)-lipgloss.Height(help)).
		Padding(1, 2).
		Render(m.entryView())

	return lipgloss.JoinVertical(lipgloss.Top, list, entry, help)
}

func (m model) listView() string {
	s := fmt.Sprintf("Total unread: %d\n", len(m.entries))
	if m.status != "" {
		s += fmt.Sprintf("Status: %s\n", m.status)
	}
	s += "\n"
	for i := 0; i < len(m.entries) && i < 5; i++ {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, m.entries[i].Title)
	}

	return s
}

func (m model) entryView() string {
	var s string
	if len(m.entries) > 0 {
		selected := m.entries[m.cursor]
		s += fmt.Sprintf("Feed: %s\n", m.feeds[selected.FeedID].Title)
		s += fmt.Sprintf("Title: %s\n", selected.Title)
		s += fmt.Sprintf("URL: %s\n\n", selected.URL)
		content, err := glamour.Render(selected.Content, "dark")
		if err != nil {
			content = fmt.Sprintf("could not render body: %v", content)
		}
		s += fmt.Sprintf("\n%s\n", content)
	}

	return s
}

func (m model) helpView() string {
	s := "Rate: 1: Not opened, 2: Not finished 3: Finished\n\n"
	s += "Press r to refresh, q to quit.\n"

	return s
}

func (m model) isVideo(feedID int64) bool {
	f, _ := m.feeds[feedID]
	catID := f.CategoryID
	if catID == CAT_VIDEO || catID == CAT_MUSIC {
		return true
	}
	return false
}
