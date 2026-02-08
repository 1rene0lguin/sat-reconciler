package ports

import "github.com/i4ene0lguin/sat-reconcilier/internal/core/domain"

type SatGateway interface {
	RequestMetadata(rfc string, start, end string, certPath, keyPath string) (string, error)
	CheckStatus(rfc, uuid string, certPath, keyPath string) (*domain.VerificationResult, error)
	DownloadPackage(rfc, packageId, certPath, keyPath string) ([]byte, error)
}
