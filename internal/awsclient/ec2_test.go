package awsclient

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"
)

func TestGetFirstJumpHostInstance(t *testing.T) {
	m := &MockEC2{IDs: []string{"i-2", "i-1"}}
	e := &EC2{client: m}
	id, err := e.GetFirstJumpHostInstance(context.Background(), "jumphost")
	require.NoError(t, err)
	// sorted alphabetically
	require.Equal(t, "i-1", id)

	m2 := &MockEC2{IDs: []string{}}
	e2 := &EC2{client: m2}
	id, err = e2.GetFirstJumpHostInstance(context.Background(), "jumphost")
	require.NoError(t, err)
	require.Equal(t, "", id)

	// Simulate error
	m3 := &MockEC2{Err: ErrSimulated}
	e3 := &EC2{client: m3}
	_, err = e3.GetFirstJumpHostInstance(context.Background(), "jumphost")
	require.Error(t, err)
}

// Dummy reuse to silence unused import for aws.
var _ aws.Config
