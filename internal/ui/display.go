package ui

import (
	"github.com/Oleexo/jumphost-cli/internal/models"
)

type Display interface {
	SetIdentity(identity Identity)
	SelectJumphostInstance(func() ([]models.JumphostInstance, error)) (models.JumphostInstance, bool, error)
	SelectService(services []models.Service) (models.Service, bool)
	SelectTarget(service models.Service, loader func() ([]models.ConnectionParams, error)) (
		models.ConnectionParams,
		bool)
	SelectRegion(regions []models.Region) models.Region
	SelectLocalPort(port int) int
	Loading(message string, loader func() (any, error)) (any, error)
	StartJumphost(jumphost models.JumphostInstance, info models.ConnectionParams) error
	PrintErrorf(format string, a ...any)
	Print(message string)
	PrintError(message string, err error)
}

type Identity interface {
	Account() string
	Arn() string
	UserID() string
}
