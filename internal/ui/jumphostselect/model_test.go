package jumphostselect

import (
	"context"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestSelectFirstInstance(t *testing.T) {
	m := NewModel(context.TODO(), []string{"i-1", "i-2"})
	items := []list.Item{item{id: "i-1"}, item{id: "i-2"}}
	m.list.SetItems(items)
	m.state = stateSelect
	// press enter
	mAny, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mAny.(Model)
	require.Equal(t, "i-1", m.result.InstanceID)
	require.False(t, m.result.Cancelled)
}

func TestCancelWithEsc(t *testing.T) {
	m := NewModel(context.TODO(), []string{"i-1"})
	items := []list.Item{item{id: "i-1"}}
	m.list.SetItems(items)
	m.state = stateSelect
	mAny, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mAny.(Model)
	require.True(t, m.result.Cancelled)
}
