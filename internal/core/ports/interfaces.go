package ports

import "github.com/1rene0lguin/sat-reconciler/internal/core/domain"

type SatGateway interface {
	RequestMetadata(rfc, start, end, downloadType, certPath, keyPath, password string) (string, error)
	CheckStatus(rfc, uuid, certPath, keyPath, password string) (*domain.VerificationResult, error)
	DownloadPackage(rfc, packageID, certPath, keyPath, password string) ([]byte, error)
}
