package tui

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Oleexo/jumphost-cli/internal/models"
	"github.com/Oleexo/jumphost-cli/internal/ui"
)

type Display struct {
	identity ui.Identity
}

func NewDisplay() *Display {
	return &Display{}
}

func (d *Display) SetIdentity(identity ui.Identity) {
	d.identity = identity
}

type jumphostItem struct {
	instance models.JumphostInstance
}

func (i jumphostItem) Title() string {
	if i.instance.Name != "" {
		return fmt.Sprintf("%s (%s)", i.instance.Name, i.instance.InstanceID)
	}
	return i.instance.InstanceID
}

func (i jumphostItem) Description() string {
	return fmt.Sprintf("IP: %s | State: %s", i.instance.PrivateIPAddress, i.instance.State)
}

func (i jumphostItem) FilterValue() string {
	return i.instance.Name + " " + i.instance.InstanceID
}

type jumphostModel struct {
	ctx       context.Context
	loader    func() ([]models.JumphostInstance, error)
	list      list.Model
	state     loadingState
	result    models.JumphostInstance
	cancelled bool
	err       error
}

type loadingState int

const (
	stateLoading loadingState = iota
	stateSelect
	stateDone
)

func newJumphostModel(ctx context.Context, loader func() ([]models.JumphostInstance, error)) jumphostModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Jumphost Instance"
	return jumphostModel{
		ctx:    ctx,
		loader: loader,
		list:   l,
		state:  stateLoading,
	}
}

func (m jumphostModel) Init() tea.Cmd {
	return tea.Batch(m.loadInstances(), m.waitCancel())
}

func (m jumphostModel) waitCancel() tea.Cmd {
	return func() tea.Msg {
		<-m.ctx.Done()
		return errMsg{m.ctx.Err()}
	}
}

type jumphostsLoadedMsg []models.JumphostInstance

type errMsg struct{ error }

func (m jumphostModel) loadInstances() tea.Cmd {
	return func() tea.Msg {
		instances, err := m.loader()
		if err != nil {
			return errMsg{err}
		}
		return jumphostsLoadedMsg(instances)
	}
}

func (m jumphostModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg
		m.cancelled = true
		m.state = stateDone
		return m, tea.Quit
	case jumphostsLoadedMsg:
		// If only one instance, select it automatically
		if len(msg) == 1 {
			m.result = msg[0]
			m.state = stateDone
			return m, tea.Quit
		}
		items := make([]list.Item, 0, len(msg))
		for _, instance := range msg {
			items = append(items, jumphostItem{instance: instance})
		}
		m.list.SetItems(items)
		m.state = stateSelect
		return m, nil
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-4)
		return m, nil
	case tea.KeyMsg:
		s := msg.String()
		if s == "ctrl+c" || s == "esc" {
			m.cancelled = true
			return m, tea.Quit
		}
		if m.state == stateSelect && s == "enter" {
			if sel, ok := m.list.SelectedItem().(jumphostItem); ok {
				m.result = sel.instance
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

func (m jumphostModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	switch m.state {
	case stateLoading:
		return lipgloss.NewStyle().Faint(true).Render("Loading jumphost instances...")
	case stateSelect:
		return m.list.View() + "\nPress Enter to select, Esc to cancel"
	case stateDone:
		return "Done"
	}
	return ""
}

func (d *Display) SelectJumphostInstance(f func() ([]models.JumphostInstance, error)) (
	models.JumphostInstance,
	bool,
	error) {
	ctx := context.Background()
	m := newJumphostModel(ctx, f)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return models.JumphostInstance{}, false, err
	}
	result := finalModel.(jumphostModel)
	return result.result, result.cancelled, result.err
}

type serviceItem struct {
	service models.Service
}

func (i serviceItem) Title() string       { return i.service.Title }
func (i serviceItem) Description() string { return i.service.Description }
func (i serviceItem) FilterValue() string { return i.service.Title }

type serviceModel struct {
	ctx       context.Context
	services  []models.Service
	list      list.Model
	state     loadingState
	result    models.Service
	cancelled bool
}

func newServiceModel(ctx context.Context, services []models.Service) serviceModel {
	items := make([]list.Item, 0, len(services))
	for _, svc := range services {
		items = append(items, serviceItem{service: svc})
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Service"
	return serviceModel{
		ctx:      ctx,
		services: services,
		list:     l,
		state:    stateSelect,
	}
}

func (m serviceModel) Init() tea.Cmd {
	return m.waitCancel()
}

func (m serviceModel) waitCancel() tea.Cmd {
	return func() tea.Msg {
		<-m.ctx.Done()
		return errMsg{m.ctx.Err()}
	}
}

func (m serviceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.cancelled = true
		m.state = stateDone
		return m, tea.Quit
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-4)
		return m, nil
	case tea.KeyMsg:
		s := msg.String()
		if s == "ctrl+c" || s == "esc" {
			m.cancelled = true
			return m, tea.Quit
		}
		if m.state == stateSelect && s == "enter" {
			if sel, ok := m.list.SelectedItem().(serviceItem); ok {
				m.result = sel.service
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

func (m serviceModel) View() string {
	switch m.state {
	case stateSelect:
		return m.list.View() + "\nPress Enter to select, Esc to cancel"
	case stateDone:
		return "Done"
	}
	return ""
}

func (d *Display) SelectService(services []models.Service) (models.Service, bool) {
	ctx := context.Background()
	m := newServiceModel(ctx, services)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return models.Service{}, true
	}
	result := finalModel.(serviceModel)
	return result.result, result.cancelled
}

type targetItem struct {
	params models.ConnectionParams
}

func (i targetItem) Title() string {
	if i.params.ClusterName != "" {
		return i.params.ClusterName
	}
	return fmt.Sprintf("%s:%d", i.params.Endpoint, i.params.RemotePort)
}

func (i targetItem) Description() string {
	desc := fmt.Sprintf("Service: %s", i.params.ServiceName)
	if i.params.Region != "" {
		desc += fmt.Sprintf(" | Region: %s", i.params.Region)
	}
	if i.params.InstanceID != "" {
		desc += fmt.Sprintf(" | Instance: %s", i.params.InstanceID)
	}
	return desc
}

func (i targetItem) FilterValue() string {
	return i.params.Endpoint + " " + i.params.ClusterName
}

type targetModel struct {
	ctx       context.Context
	loader    func() ([]models.ConnectionParams, error)
	service   models.Service
	list      list.Model
	state     loadingState
	result    models.ConnectionParams
	cancelled bool
	err       error
}

func newTargetModel(
	ctx context.Context,
	service models.Service,
	loader func() ([]models.ConnectionParams, error)) targetModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = fmt.Sprintf("Select %s Target", service.Title)
	return targetModel{
		ctx:     ctx,
		loader:  loader,
		service: service,
		list:    l,
		state:   stateLoading,
	}
}

func (m targetModel) Init() tea.Cmd {
	return tea.Batch(m.loadTargets(), m.waitCancel())
}

func (m targetModel) waitCancel() tea.Cmd {
	return func() tea.Msg {
		<-m.ctx.Done()
		return errMsg{m.ctx.Err()}
	}
}

type targetsLoadedMsg []models.ConnectionParams

func (m targetModel) loadTargets() tea.Cmd {
	return func() tea.Msg {
		targets, err := m.loader()
		if err != nil {
			return errMsg{err}
		}
		return targetsLoadedMsg(targets)
	}
}

func (m targetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg
		m.cancelled = true
		m.state = stateDone
		return m, tea.Quit
	case targetsLoadedMsg:
		items := make([]list.Item, 0, len(msg))
		for _, target := range msg {
			items = append(items, targetItem{params: target})
		}
		m.list.SetItems(items)
		m.state = stateSelect
		return m, nil
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-4)
		return m, nil
	case tea.KeyMsg:
		s := msg.String()
		if s == "ctrl+c" || s == "esc" {
			m.cancelled = true
			return m, tea.Quit
		}
		if m.state == stateSelect && s == "enter" {
			if sel, ok := m.list.SelectedItem().(targetItem); ok {
				m.result = sel.params
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

func (m targetModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	switch m.state {
	case stateLoading:
		return lipgloss.NewStyle().Faint(true).Render("Loading targets...")
	case stateSelect:
		return m.list.View() + "\nPress Enter to select, Esc to cancel"
	case stateDone:
		return "Done"
	}
	return ""
}

func (d *Display) SelectTarget(
	service models.Service,
	loader func() ([]models.ConnectionParams, error)) (models.ConnectionParams, bool) {
	ctx := context.Background()
	m := newTargetModel(ctx, service, loader)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return models.ConnectionParams{}, true
	}
	result := finalModel.(targetModel)
	return result.result, result.cancelled
}

type regionItem struct {
	region models.Region
}

func (i regionItem) Title() string       { return i.region.Name }
func (i regionItem) Description() string { return i.region.Code }
func (i regionItem) FilterValue() string { return i.region.Name + " " + i.region.Code }

type regionModel struct {
	ctx       context.Context
	regions   []models.Region
	list      list.Model
	state     loadingState
	result    models.Region
	cancelled bool
}

func newRegionModel(ctx context.Context, regions []models.Region) regionModel {
	items := make([]list.Item, 0, len(regions))
	for _, region := range regions {
		items = append(items, regionItem{region: region})
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Region"
	return regionModel{
		ctx:     ctx,
		regions: regions,
		list:    l,
		state:   stateSelect,
	}
}

func (m regionModel) Init() tea.Cmd {
	return m.waitCancel()
}

func (m regionModel) waitCancel() tea.Cmd {
	return func() tea.Msg {
		<-m.ctx.Done()
		return errMsg{m.ctx.Err()}
	}
}

func (m regionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.cancelled = true
		m.state = stateDone
		return m, tea.Quit
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-4)
		return m, nil
	case tea.KeyMsg:
		s := msg.String()
		if s == "ctrl+c" || s == "esc" {
			m.cancelled = true
			return m, tea.Quit
		}
		if m.state == stateSelect && s == "enter" {
			if sel, ok := m.list.SelectedItem().(regionItem); ok {
				m.result = sel.region
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

func (m regionModel) View() string {
	switch m.state {
	case stateSelect:
		return m.list.View() + "\nPress Enter to select, Esc to cancel"
	case stateDone:
		return "Done"
	}
	return ""
}

func (d *Display) SelectRegion(regions []models.Region) models.Region {
	ctx := context.Background()
	m := newRegionModel(ctx, regions)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		// Return first region as fallback
		if len(regions) > 0 {
			return regions[0]
		}
		return models.Region{}
	}
	result := finalModel.(regionModel)
	if result.cancelled && len(regions) > 0 {
		return regions[0]
	}
	return result.result
}

type portInputModel struct {
	ctx         context.Context
	defaultPort int
	input       textinput.Model
	state       loadingState
	result      int
	cancelled   bool
}

func newPortInputModel(ctx context.Context, defaultPort int) portInputModel {
	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("%d", defaultPort)
	ti.CharLimit = 6
	ti.Focus()
	return portInputModel{
		ctx:         ctx,
		defaultPort: defaultPort,
		input:       ti,
		state:       stateSelect,
		result:      defaultPort,
	}
}

func (m portInputModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.waitCancel())
}

func (m portInputModel) waitCancel() tea.Cmd {
	return func() tea.Msg {
		<-m.ctx.Done()
		return errMsg{m.ctx.Err()}
	}
}

func (m portInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case errMsg:
		m.cancelled = true
		m.state = stateDone
		return m, tea.Quit
	case tea.KeyMsg:
		s := msg.String()
		if s == "ctrl+c" || s == "esc" {
			m.cancelled = true
			return m, tea.Quit
		}
		if s == "enter" {
			if m.input.Value() == "" {
				m.result = m.defaultPort
			} else {
				if port, err := strconv.Atoi(m.input.Value()); err == nil && port > 0 && port <= 65535 {
					m.result = port
				} else {
					m.input.SetValue("")
					return m, nil
				}
			}
			m.state = stateDone
			return m, tea.Quit
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m portInputModel) View() string {
	if m.state == stateDone {
		return "Done"
	}
	return fmt.Sprintf(
		"Enter local port (default: %d):\n\n%s\n\n%s",
		m.defaultPort,
		m.input.View(),
		lipgloss.NewStyle().Faint(true).Render("Press Enter to confirm, Esc to cancel"),
	)
}

func (d *Display) SelectLocalPort(port int) int {
	ctx := context.Background()
	m := newPortInputModel(ctx, port)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return port
	}
	result := finalModel.(portInputModel)
	if result.cancelled {
		return port
	}
	return result.result
}

type loadingModel struct {
	ctx     context.Context
	message string
	loader  func() (any, error)
	spinner spinner.Model
	result  any
	err     error
	done    bool
}

func newLoadingModel(ctx context.Context, message string, loader func() (any, error)) loadingModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return loadingModel{
		ctx:     ctx,
		message: message,
		loader:  loader,
		spinner: s,
	}
}

func (m loadingModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.load(), m.waitCancel())
}

func (m loadingModel) waitCancel() tea.Cmd {
	return func() tea.Msg {
		<-m.ctx.Done()
		return errMsg{m.ctx.Err()}
	}
}

type loadCompleteMsg struct {
	result any
	err    error
}

func (m loadingModel) load() tea.Cmd {
	return func() tea.Msg {
		result, err := m.loader()
		return loadCompleteMsg{result: result, err: err}
	}
}

func (m loadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg
		m.done = true
		return m, tea.Quit
	case loadCompleteMsg:
		m.result = msg.result
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m loadingModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), m.message)
}

func (d *Display) Loading(message string, loader func() (any, error)) (any, error) {
	ctx := context.Background()
	m := newLoadingModel(ctx, message, loader)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	result := finalModel.(loadingModel)
	return result.result, result.err
}

func (d *Display) StartJumphost(jumphost models.JumphostInstance, info models.ConnectionParams) error {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("2"))

	header := style.Render("Starting SSH tunnel...")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, header)
	fmt.Fprintf(os.Stdout, "  Jumphost: %s (%s)\n", jumphost.Name, jumphost.InstanceID)
	fmt.Fprintf(os.Stdout, "  Target: %s:%d\n", info.Endpoint, info.RemotePort)
	fmt.Fprintf(os.Stdout, "  Local Port: %d\n", info.LocalPort)
	if info.ApplyHosts {
		fmt.Fprintf(os.Stdout, "  Hosts Entry: %s -> 127.0.0.1\n", info.Endpoint)
	}
	fmt.Fprintln(os.Stdout)

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))
	fmt.Fprintln(os.Stdout, successStyle.Render("✓ Connecting..."))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, lipgloss.NewStyle().Faint(true).Render("Press Ctrl+C to terminate..."))
	fmt.Fprintln(os.Stdout)

	return jumphost.Start(context.Background(), info)
}

var (
	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9"))
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))
)

func (d *Display) PrintErrorf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, errorStyle.Render("Error: ")+msg)
}

func (d *Display) Print(message string) {
	fmt.Fprintln(os.Stdout, infoStyle.Render(message))
}

func (d *Display) PrintError(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s %s: %v\n",
		errorStyle.Render("Error"),
		message,
		err,
	)
}
