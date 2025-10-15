package awsclient

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"
)

func TestListEndpoints(t *testing.T) {
	mock := &MockRDS{Endpoints: []Endpoint{{Address: "b.example", Port: 3306}, {Address: "a.example", Port: 5432}}}
	r := &RDS{client: mock}
	eps, err := r.ListEndpoints(context.Background())
	require.NoError(t, err)
	require.Len(t, eps, 2)
	require.Equal(t, "a.example", eps[0].Address)
}

// Dummy reuse to silence unused import for aws.
var _ = aws.Config{}
