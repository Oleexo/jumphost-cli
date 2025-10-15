package jumphostselect

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Result of jumphost selection.
type Result struct {
	InstanceID string
	Cancelled  bool
}

type state int

const (
	stateLoading state = iota
	stateSelect
	stateDone
)

type item struct{ id string }

func (i item) Title() string       { return i.id }
func (i item) Description() string { return "EC2 jumphost instance" }
func (i item) FilterValue() string { return i.id }

// Model for selecting jumphost instance.
type Model struct {
	ctx    context.Context
	state  state
	list   list.Model
	result Result
	ids    []string
	err    error
}

// NewModel creates a new selection model.
func NewModel(ctx context.Context, ids []string) Model {
	m := Model{ctx: ctx, state: stateLoading, ids: ids}
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Jumphost Instance"
	m.list = l
	return m
}

func (m Model) Init() tea.Cmd { return tea.Batch(m.load(), m.waitCancel()) }

func (m Model) waitCancel() tea.Cmd {
	return func() tea.Msg { <-m.ctx.Done(); return errMsg{m.ctx.Err()} }
}

func (m Model) load() tea.Cmd {
	return func() tea.Msg {
		items := make([]list.Item, 0, len(m.ids))
		for _, id := range m.ids {
			items = append(items, item{id: id})
		}
		return itemsMsg(items)
	}
}

type itemsMsg []list.Item

type errMsg struct{ error }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.result.Cancelled = true
		m.err = msg
		m.state = stateDone
		return m, tea.Quit
	case itemsMsg:
		m.list.SetItems(msg)
		m.state = stateSelect
		return m, nil
	case tea.KeyMsg:
		s := msg.String()
		if s == "ctrl+c" || s == "esc" {
			m.result.Cancelled = true
			return m, tea.Quit
		}
		if m.state == stateSelect && s == "enter" {
			if sel, ok := m.list.SelectedItem().(item); ok {
				m.result.InstanceID = sel.id
				m.state = stateDone
				return m, tea.Quit
			}
		}
	}
	if m.state == stateSelect {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	switch m.state {
	case stateLoading:
		return lipgloss.NewStyle().Faint(true).Render("Loading jumphost instances...")
	case stateSelect:
		return m.list.View() + "\nEnter to select, Esc to cancel"
	case stateDone:
		return "Done"
	}
	return ""
}

// Run executes program.
func Run(m Model) (Result, error) {
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	return finalModel.(Model).result, nil
}
