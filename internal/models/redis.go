package models

type Redis struct {
	CacheClusterID     string
	Endpoint           string
	Port               int
	CacheNodeType      string
	Engine             string
	EngineVersion      string
	CacheClusterStatus string
}
