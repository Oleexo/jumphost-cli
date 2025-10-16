package models

type Redshift struct {
	ClusterIdentifier string
	Endpoint          string
	Port              int
	ClusterStatus     string
	NodeType          string
	NumberOfNodes     int
}
