package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseEndpoint(t *testing.T) {
	h, p, err := parseEndpoint("db.example:5432")
	require.NoError(t, err)
	require.Equal(t, "db.example", h)
	require.Equal(t, 5432, p)
	_, _, err = parseEndpoint("bad")
	require.Error(t, err)
}
