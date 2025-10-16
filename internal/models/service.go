package models

type Service struct {
	Key         ServiceType
	Title       string
	Description string
}

func (s Service) Name() string {
	return s.Title
}

type ServiceType string

func (s ServiceType) String() string {
	return string(s)
}

const (
	ServiceTypeNone       ServiceType = ""
	ServiceTypeRDS        ServiceType = "rds"
	ServiceTypeOpenSearch ServiceType = "opensearch"
	ServiceTypeRedshift   ServiceType = "redshift"
	ServiceTypeRedis      ServiceType = "redis"
	ServiceTypeDocumentDB ServiceType = "documentdb"
	ServiceTypeEKS        ServiceType = "eks"
)
