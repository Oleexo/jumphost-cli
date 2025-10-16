package awsclient

import (
	"context"
	"sort"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/opensearch/types"
)

// OpenSearchAPI abstraction.
type OpenSearchAPI interface {
	ListDomainNames(
		ctx context.Context,
		params *opensearch.ListDomainNamesInput,
		optFns ...func(*opensearch.Options)) (*opensearch.ListDomainNamesOutput, error)
	DescribeDomain(
		ctx context.Context,
		params *opensearch.DescribeDomainInput,
		optFns ...func(*opensearch.Options)) (*opensearch.DescribeDomainOutput, error)
}

type OpenSearch struct{ client OpenSearchAPI }

func NewOpenSearch(cfg awsv2.Config) *OpenSearch {
	return &OpenSearch{client: opensearch.NewFromConfig(cfg)}
}

// Domain describes an OpenSearch domain endpoint.
type Domain struct {
	DomainName string
	Endpoint   string
	Port       int
	EngineType string
}

// ListDomains lists endpoints of all OpenSearch domains.
func (o *OpenSearch) ListDomains(ctx context.Context) ([]Domain, error) {
	var domains []Domain

	// List all domain names
	listOut, err := o.client.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		return nil, err
	}

	// Get details for each domain
	for _, domainInfo := range listOut.DomainNames {
		if domainInfo.DomainName == nil {
			continue
		}

		descOut, err := o.client.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
			DomainName: domainInfo.DomainName,
		})
		if err != nil {
			continue // Skip domains we can't describe
		}

		if descOut.DomainStatus != nil && descOut.DomainStatus.Endpoint != nil {
			domain := Domain{
				DomainName: *domainInfo.DomainName,
				Endpoint:   *descOut.DomainStatus.Endpoint,
				Port:       443, // OpenSearch uses HTTPS (port 443)
			}
			if domainInfo.EngineType != "" {
				domain.EngineType = string(domainInfo.EngineType)
			}
			domains = append(domains, domain)
		}
	}

	sort.Slice(domains, func(i, j int) bool {
		return domains[i].DomainName < domains[j].DomainName
	})

	return domains, nil
}

// Mock for tests.
type MockOpenSearch struct {
	Domains []Domain
	Err     error
}

func (m *MockOpenSearch) ListDomainNames(
	ctx context.Context,
	params *opensearch.ListDomainNamesInput,
	optFns ...func(*opensearch.Options)) (*opensearch.ListDomainNamesOutput, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	out := &opensearch.ListDomainNamesOutput{}
	for _, d := range m.Domains {
		name := d.DomainName
		engineType := types.EngineType(d.EngineType)
		out.DomainNames = append(out.DomainNames, types.DomainInfo{
			DomainName: &name,
			EngineType: engineType,
		})
	}
	return out, nil
}

func (m *MockOpenSearch) DescribeDomain(
	ctx context.Context,
	params *opensearch.DescribeDomainInput,
	optFns ...func(*opensearch.Options)) (*opensearch.DescribeDomainOutput, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	for _, d := range m.Domains {
		if d.DomainName == *params.DomainName {
			endpoint := d.Endpoint
			return &opensearch.DescribeDomainOutput{
				DomainStatus: &types.DomainStatus{
					Endpoint: &endpoint,
				},
			}, nil
		}
	}
	return &opensearch.DescribeDomainOutput{}, nil
}
