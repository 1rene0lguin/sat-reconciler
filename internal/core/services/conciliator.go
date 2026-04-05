package services

import (
	"fmt"

	"github.com/1rene0lguin/sat-reconciler/internal/core/domain"
	"github.com/1rene0lguin/sat-reconciler/internal/core/ports"
)

const (
	msgRequestFinished  = "Solicitud Terminada. Encontrados %d paquetes."
	msgDownloadingPkg   = "\n -> Descargando paquete: %s ... OK"
	msgRequestInProcess = "El SAT sigue procesando tu solicitud (Estado: En Proceso). Intenta más tarde."
	msgStatusFormat     = "Estado: %s - %s"
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

func (s *ConciliatorService) VerifyRequest(rfc, uuid, cert, key, password string) (string, error) {
	result, err := s.satGateway.CheckStatus(rfc, uuid, cert, key, password)
	if err != nil {
		return "", err
	}

	if result.Status == domain.StatusFinished {
		msg := fmt.Sprintf(msgRequestFinished, len(result.PackageIDs))

		for _, pkgID := range result.PackageIDs {
			msg += fmt.Sprintf(msgDownloadingPkg, pkgID)
		}
		return msg, nil
	}

	if result.Status == domain.StatusInProcess || result.Status == domain.StatusAccepted {
		return msgRequestInProcess, nil
	}

	return fmt.Sprintf(msgStatusFormat, result.Status, result.Message), nil
}

func (s *ConciliatorService) RequestMetadata(rfc, start, end, downloadType, certPath, keyPath, password string) (string, error) {
	return s.satGateway.RequestMetadata(rfc, start, end, downloadType, certPath, keyPath, password)
}

func (s *ConciliatorService) DownloadPackage(rfc, pkgID, cert, key, password string) ([]byte, error) {
	return s.satGateway.DownloadPackage(rfc, pkgID, cert, key, password)
}

func (s *ConciliatorService) CheckStatus(rfc, uuid, cert, key, password string) (*domain.VerificationResult, error) {
	return s.satGateway.CheckStatus(rfc, uuid, cert, key, password)
}
