package connectflow

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/list"
	textinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oleexo/jumphost-cli/internal/awsclient"
)

// Result returned after the UI exits.
type Result struct {
	Endpoint   string
	RemotePort int
	LocalPort  int
	ApplyHosts bool
	Cancelled  bool
}

// Model implements the selection flow: list endpoints -> input local port -> confirm hosts mapping.
// To keep it compact, we use states.

type state int

const (
	stateLoading state = iota
	stateSelect
	stateLocalPort
	stateConfirmHosts
	stateDone
)

type item struct {
	title, desc string
	endpoint    awsclient.Endpoint
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Model struct.
type Model struct {
	client interface {
		ListEndpoints(context.Context) ([]awsclient.Endpoint, error)
	}
	state     state
	list      list.Model
	portInput textinput.Model
	err       error
	result    Result
	ctx       context.Context
	acctInfo  string // short identity display
}

// NewModel now takes a parent context so cancellation (SIGINT etc.) stops loading.
// identity is optional; if provided, shown in the list title.
func NewModel(ctx context.Context, rds *awsclient.RDS, identity awsclient.Identity) Model {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	title := "Select RDS Endpoint"
	if identity.Account != "" || identity.UserID != "" {
		shortUser := identity.UserID
		if len(shortUser) > 28 { // truncate for nicer title
			shortUser = shortUser[:25] + "..."
		}
		// Keep it compact; account first (helps when switching accounts)
		title = fmt.Sprintf("Acct %s User %s - Select RDS Endpoint", identity.Account, shortUser)
	}
	l.Title = title
	pi := textinput.New()
	pi.Placeholder = "Local port (default remote)"
	pi.CharLimit = 6
	return Model{
		client: rds, state: stateLoading, list: l, portInput: pi, ctx: ctx,
		result: Result{ApplyHosts: true}, acctInfo: identity.String(),
	}
}

// Init loads endpoints asynchronously.
func (m Model) Init() tea.Cmd { return tea.Batch(m.loadEndpoints(), m.waitForCancel()) }

func (m Model) waitForCancel() tea.Cmd {
	return func() tea.Msg {
		<-m.ctx.Done()
		return errMsg{m.ctx.Err()}
	}
}

func (m Model) loadEndpoints() tea.Cmd {
	return func() tea.Msg {
		endpoints, err := m.client.ListEndpoints(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		items := make([]list.Item, 0, len(endpoints))
		for _, ep := range endpoints {
			title := fmt.Sprintf("%s:%d", ep.Address, ep.Port)
			// Build a descriptive label with DB identifier, engine, status, and instance class
			desc := ""
			if ep.DBInstanceID != "" {
				desc = ep.DBInstanceID
			}
			if ep.Engine != "" {
				if desc != "" {
					desc += " | "
				}
				desc += ep.Engine
				if ep.EngineVersion != "" {
					desc += " " + ep.EngineVersion
				}
			}
			if ep.DBInstanceStatus != "" {
				if desc != "" {
					desc += " | "
				}
				desc += ep.DBInstanceStatus
			}
			if ep.DBInstanceClass != "" {
				if desc != "" {
					desc += " | "
				}
				desc += ep.DBInstanceClass
			}
			if desc == "" {
				desc = "RDS endpoint"
			}
			items = append(items,
				item{title: title, desc: desc, endpoint: ep})
		}
		return endpointsMsg(items)
	}
}

type endpointsMsg []list.Item

type errMsg struct{ error }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Set list dimensions based on terminal size
		h, v := lipgloss.NewStyle().GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-4) // -4 for footer text
		return m, nil
	case errMsg:
		if msg.error == context.Canceled {
			m.result.Cancelled = true
		}
		m.err = msg
		m.state = stateDone
		return m, tea.Quit
	case endpointsMsg:
		m.list.SetItems(msg)
		m.state = stateSelect
		return m, nil
	case tea.KeyMsg:
		s := msg.String()
		// global cancel
		if s == "ctrl+c" || s == "esc" {
			m.result.Cancelled = true
			return m, tea.Quit
		}
		switch m.state {
		case stateSelect:
			if s == "enter" {
				if sel, ok := m.list.SelectedItem().(item); ok {
					m.result.Endpoint = sel.endpoint.Address
					m.result.RemotePort = sel.endpoint.Port
					m.state = stateLocalPort
					return m, nil
				}
			}
		case stateLocalPort:
			if s == "enter" {
				if m.portInput.Value() == "" {
					m.result.LocalPort = m.result.RemotePort
				} else {
					p, err := strconv.Atoi(m.portInput.Value())
					if err == nil {
						m.result.LocalPort = p
					} else {
						m.portInput.SetValue("")
						return m, nil
					}
				}
				m.state = stateConfirmHosts
				return m, nil
			}
		case stateConfirmHosts:
			if s == "y" || s == "enter" {
				m.result.ApplyHosts = true
				m.state = stateDone
				return m, tea.Quit
			}
			if s == "n" {
				m.result.ApplyHosts = false
				m.state = stateDone
				return m, tea.Quit
			}
		}
	}

	switch m.state {
	case stateSelect:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case stateLocalPort:
		var cmd tea.Cmd
		m.portInput, cmd = m.portInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	header := ""
	if m.acctInfo != "" {
		header = lipgloss.NewStyle().Faint(true).Render(m.acctInfo) + "\n"
	}
	switch m.state {
	case stateLoading:
		return header + lipgloss.NewStyle().Faint(true).Render("Loading RDS endpoints...")
	case stateSelect:
		return header + m.list.View() + "\nPress Enter to select, Esc to cancel"
	case stateLocalPort:
		return header + fmt.Sprintf("Selected %s:%d\nEnter local port (blank to use same):\n%s", m.result.Endpoint,
			m.result.RemotePort, m.portInput.View())
	case stateConfirmHosts:
		return header + fmt.Sprintf("Map %s to 127.0.0.1 in /etc/hosts? (y/N)", m.result.Endpoint)
	case stateDone:
		return header + "Done"
	}
	return header
}

// Run executes the program and returns Result.
func Run(m Model) (Result, error) {
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	res := finalModel.(Model).result
	return res, nil
}

// Small timeout helper removed; relying on parent context. Keep a no-op ref to time for import.
var _ = time.Second
