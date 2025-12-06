package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type responseMsg struct{}

func listenForActivity(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		for {
			time.Sleep(time.Millisecond * time.Duration(rand.Int63n(900)+100))
			sub <- struct{}{}
		}
	}
}

func waitForActivity(feed chan MFResult) tea.Cmd {
	return func() tea.Msg {
		return <-feed
	}
}

type model struct {
	entries   []string
	cursor    int
	selected  map[int]struct{}
	feed      chan MFResult
	responses int
	spinner   spinner.Model
	done      chan bool
	quitting  bool
}

func InitialModel(feed chan MFResult, done chan bool) model {
	return model{
		feed:     feed,
		done:     done,
		entries:  make([]string, 0),
		selected: make(map[int]struct{}),
		spinner:  spinner.New(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForActivity(m.feed),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case responseMsg:
		m.responses++
		return m, waitForActivity(m.feed)
	case MFResult:
		entries := make([]string, 0)
		for _, entry := range msg.Entries {
			entries = append(entries, entry.Title)
		}
		m.entries = entries
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.done <- true
			m.quitting = true
			return m, tea.Quit
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
	s := fmt.Sprintf("\n %s Events received: %d\n\n", m.spinner.View(), m.responses)
	if m.quitting {
		s += "\n"
	}
	for i, choice := range m.entries {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}
	s += "\nPress q to quit.\n"

	return s
}
