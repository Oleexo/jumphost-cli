package connectflow

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/Oleexo/jumphost-cli/internal/awsclient"
)

type mockRDSList struct{ eps []awsclient.Endpoint }

func (m mockRDSList) ListEndpoints(_ context.Context) ([]awsclient.Endpoint, error) {
	return m.eps, nil
}

// Test that the model initializes and exits quickly using a short timeout context.
func TestModelInit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	m := NewModel(ctx, &awsclient.RDS{}, awsclient.Identity{})
	m.client = mockRDSList{eps: []awsclient.Endpoint{{Address: "db.example", Port: 5432}}}
	start := time.Now()
	prog := tea.NewProgram(m, tea.WithoutRenderer(), tea.WithInput(nil))
	_, err := prog.Run()
	require.NoError(t, err)
	require.Less(t, time.Since(start), 2*time.Second, "program should exit quickly")
}
