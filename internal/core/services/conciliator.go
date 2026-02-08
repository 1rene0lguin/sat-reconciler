package services

import (
	"fmt"

	"github.com/i4ene0lguin/sat-reconcilier/internal/core/domain"
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
	result, err := s.satGateway.CheckStatus(rfc, uuid, cert, key)
	if err != nil {
		return "", err
	}

	if result.Status == domain.StatusFinished {
		msg := fmt.Sprintf("Solicitud Terminada. Encontrados %d paquetes.", len(result.PackageIDs))

		for _, pkgID := range result.PackageIDs {
			msg += fmt.Sprintf("\n -> Descargando paquete: %s ... OK", pkgID)
		}
		return msg, nil
	}

	if result.Status == domain.StatusInProcess || result.Status == domain.StatusAccepted {
		return "El SAT sigue procesando tu solicitud (Estado: En Proceso). Intenta más tarde.", nil
	}

	return fmt.Sprintf("Estado: %s - %s", result.Status, result.Message), nil
}
