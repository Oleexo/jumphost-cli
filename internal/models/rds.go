package models

type RDS struct {
	DBInstanceID     string
	Address          string
	Port             int
	Engine           string
	EngineVersion    string
	DBInstanceStatus string
	DBInstanceClass  string
}
