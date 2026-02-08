package ports

import "github.com/i4ene0lguin/sat-reconcilier/internal/core/domain"

type SatGateway interface {
	RequestMetadata(rfc, start, end, certPath, keyPath string) (string, error)
	CheckStatus(rfc, uuid, certPath, keyPath string) (*domain.VerificationResult, error)
	DownloadPackage(rfc, packageID, certPath, keyPath string) ([]byte, error)
}
