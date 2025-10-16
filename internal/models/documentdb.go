package models

type DocumentDB struct {
	ClusterIdentifier string
	Endpoint          string
	Port              int
	Engine            string
	EngineVersion     string
	Status            string
}
