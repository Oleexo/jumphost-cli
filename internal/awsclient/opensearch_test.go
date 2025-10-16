package awsclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenSearch(t *testing.T) {
	// Just verify construction doesn't panic
	// Can't fully test without valid AWS config
	// os := NewOpenSearch(aws.Config{})
	// assert.NotNil(t, os)
}

func TestOpenSearch_ListDomains(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns domains successfully", func(t *testing.T) {
		mockClient := &MockOpenSearch{
			Domains: []Domain{
				{
					DomainName: "domain-1",
					Endpoint:   "domain-1.us-east-1.es.amazonaws.com",
					Port:       443,
					EngineType: "OpenSearch",
				},
				{
					DomainName: "domain-2",
					Endpoint:   "domain-2.us-east-1.es.amazonaws.com",
					Port:       443,
					EngineType: "Elasticsearch",
				},
			},
		}

		os := &OpenSearch{client: mockClient}
		domains, err := os.ListDomains(ctx)

		require.NoError(t, err)
		assert.Len(t, domains, 2)
		assert.Equal(t, "domain-1", domains[0].DomainName)
		assert.Equal(t, "domain-1.us-east-1.es.amazonaws.com", domains[0].Endpoint)
		assert.Equal(t, 443, domains[0].Port)
		assert.Equal(t, "OpenSearch", domains[0].EngineType)
	})

	t.Run("Returns error when API fails", func(t *testing.T) {
		mockClient := &MockOpenSearch{
			Err: fmt.Errorf("API error"),
		}

		os := &OpenSearch{client: mockClient}
		domains, err := os.ListDomains(ctx)

		assert.Error(t, err)
		assert.Nil(t, domains)
	})

	t.Run("Returns empty list when no domains", func(t *testing.T) {
		mockClient := &MockOpenSearch{
			Domains: []Domain{},
		}

		os := &OpenSearch{client: mockClient}
		domains, err := os.ListDomains(ctx)

		require.NoError(t, err)
		assert.Empty(t, domains)
	})

	t.Run("Sorts domains by name", func(t *testing.T) {
		mockClient := &MockOpenSearch{
			Domains: []Domain{
				{
					DomainName: "zebra-domain",
					Endpoint:   "zebra.us-east-1.es.amazonaws.com",
					Port:       443,
				},
				{
					DomainName: "alpha-domain",
					Endpoint:   "alpha.us-east-1.es.amazonaws.com",
					Port:       443,
				},
				{
					DomainName: "beta-domain",
					Endpoint:   "beta.us-east-1.es.amazonaws.com",
					Port:       443,
				},
			},
		}

		os := &OpenSearch{client: mockClient}
		domains, err := os.ListDomains(ctx)

		require.NoError(t, err)
		assert.Len(t, domains, 3)
		assert.Equal(t, "alpha-domain", domains[0].DomainName)
		assert.Equal(t, "beta-domain", domains[1].DomainName)
		assert.Equal(t, "zebra-domain", domains[2].DomainName)
	})
}

func TestMockOpenSearch_ListDomainNames(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns mocked domain names", func(t *testing.T) {
		mock := &MockOpenSearch{
			Domains: []Domain{
				{
					DomainName: "test-domain",
					Endpoint:   "test.us-east-1.es.amazonaws.com",
					Port:       443,
					EngineType: "OpenSearch",
				},
			},
		}

		out, err := mock.ListDomainNames(ctx, nil)

		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Len(t, out.DomainNames, 1)
		assert.Equal(t, "test-domain", *out.DomainNames[0].DomainName)
	})

	t.Run("Returns error when configured", func(t *testing.T) {
		mock := &MockOpenSearch{
			Err: fmt.Errorf("mock error"),
		}

		out, err := mock.ListDomainNames(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, out)
	})
}

func TestMockOpenSearch_DescribeDomain(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns domain details", func(t *testing.T) {
		domainName := "test-domain"
		mock := &MockOpenSearch{
			Domains: []Domain{
				{
					DomainName: "test-domain",
					Endpoint:   "test.us-east-1.es.amazonaws.com",
					Port:       443,
				},
			},
		}

		// Use the actual opensearch input type
		out, err := mock.DescribeDomain(ctx, &opensearch.DescribeDomainInput{DomainName: &domainName})

		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotNil(t, out.DomainStatus)
		assert.Equal(t, "test.us-east-1.es.amazonaws.com", *out.DomainStatus.Endpoint)
	})

	t.Run("Returns error when configured", func(t *testing.T) {
		domainName := "test-domain"
		mock := &MockOpenSearch{
			Err: fmt.Errorf("mock error"),
		}

		out, err := mock.DescribeDomain(ctx, &opensearch.DescribeDomainInput{DomainName: &domainName})

		assert.Error(t, err)
		assert.Nil(t, out)
	})
}
