package services

import (
	"fmt"

	"github.com/i4ene0lguin/sat-reconcilier/internal/core/ports"
)

type ConciliatorService struct {
	satGateway ports.SatGateway
	// storage ports.Repository (Futuro)
}

func NewConciliatorService(gateway ports.SatGateway) *ConciliatorService {
	return &ConciliatorService{
		satGateway: gateway,
	}
}

func (s *ConciliatorService) VerifyRequest(rfc, uuid, cert, key string) (string, error) {
	if uuid == "" {
		return "", fmt.Errorf("UUID inválido")
	}

	result, err := s.satGateway.CheckStatus(rfc, uuid, cert, key)
	if err != nil {
		return "", err
	}

	switch result.Status {
	case 3:
		return "¡Listo para descargar!", nil
	case 2:
		return "Sigue cocinándose...", nil
	default:
		return result.Message, nil
	}
}
