package connectflow

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/Oleexo/jumphost-cli/internal/awsclient"
)

type mockRDS struct{ eps []awsclient.Endpoint }

func (m mockRDS) ListEndpoints(ctx context.Context) ([]awsclient.Endpoint, error) { return m.eps, nil }

func TestFullFlow_NoHosts(t *testing.T) {
	ctx := context.Background()
	m := NewModel(ctx, &awsclient.RDS{}, awsclient.Identity{})
	m.client = mockRDS{eps: []awsclient.Endpoint{{Address: "db.example", Port: 5432}}}
	// load endpoints directly
	msg := m.loadEndpoints()()
	mAny, _ := m.Update(msg)
	m = mAny.(Model)
	// select
	mAny, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mAny.(Model)
	// set port directly (simulate typing)
	m.portInput.SetValue("15432")
	mAny, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // advance to confirm hosts
	m = mAny.(Model)
	mAny, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = mAny.(Model)
	require.Equal(t, 15432, m.result.LocalPort)
}

func TestFullFlow_DefaultLocalPortAndApplyHosts(t *testing.T) {
	ctx := context.Background()
	m := NewModel(ctx, &awsclient.RDS{}, awsclient.Identity{})
	m.client = mockRDS{eps: []awsclient.Endpoint{{Address: "db2.example", Port: 3306}}}
	msg := m.loadEndpoints()()
	mAny, _ := m.Update(msg)
	m = mAny.(Model)
	mAny, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // select
	m = mAny.(Model)
	mAny, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // blank local port
	m = mAny.(Model)
	mAny, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm hosts
	m = mAny.(Model)
	require.Equal(t, 3306, m.result.LocalPort)
	require.True(t, m.result.ApplyHosts)
}

func TestCancellationMessage(t *testing.T) {
	m := NewModel(context.Background(), &awsclient.RDS{}, awsclient.Identity{})
	// send cancellation error message directly
	mAny, _ := m.Update(errMsg{context.Canceled})
	m = mAny.(Model)
	require.True(t, m.result.Cancelled)
}
