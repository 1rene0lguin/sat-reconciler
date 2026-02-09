package services_test

import (
	"testing"

	"github.com/1rene0lguin/sat-reconciler/internal/core/domain"
	"github.com/1rene0lguin/sat-reconciler/internal/core/services"
)

// MockSatGateway simula al SAT para no depender de internet en los tests
type MockSatGateway struct {
	RequestMetadataFunc func(rfc, start, end, c, k string) (string, error)
	CheckStatusFunc     func(rfc, uuid, c, k string) (*domain.VerificationResult, error)
	DownloadPackageFunc func(rfc, id, c, k string) ([]byte, error)
}

func (m *MockSatGateway) RequestMetadata(rfc, start, end, c, k string) (string, error) {
	if m.RequestMetadataFunc != nil {
		return m.RequestMetadataFunc(rfc, start, end, c, k)
	}
	return "UUID-TEST-123", nil
}

func (m *MockSatGateway) CheckStatus(rfc, uuid, c, k string) (*domain.VerificationResult, error) {
	if m.CheckStatusFunc != nil {
		return m.CheckStatusFunc(rfc, uuid, c, k)
	}
	// Default behavior
	return &domain.VerificationResult{
		UUID:       uuid,
		Status:     domain.StatusFinished,
		Message:    "Terminado Correctamente",
		PackageIDs: []string{"PKG-A", "PKG-B"},
	}, nil
}

func (m *MockSatGateway) DownloadPackage(rfc, id, c, k string) ([]byte, error) {
	if m.DownloadPackageFunc != nil {
		return m.DownloadPackageFunc(rfc, id, c, k)
	}
	return []byte("CONTENIDO-ZIP-MOCK"), nil
}

func TestVerifyRequest_FinishedStatus(t *testing.T) {
	// 1. Arrange
	mockGateway := &MockSatGateway{}
	service := services.NewConciliatorService(mockGateway)

	// 2. Act
	result, err := service.VerifyRequest("RFC123", "UUID-TEST", "dummy.cer", "dummy.key")

	// 3. Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !contains(result, "Solicitud Terminada") {
		t.Errorf("Expected message to contain 'Solicitud Terminada', got: %s", result)
	}
}

func TestVerifyRequest_InProcessStatus(t *testing.T) {
	// 1. Arrange
	mockGateway := &MockSatGateway{
		CheckStatusFunc: func(rfc, uuid, c, k string) (*domain.VerificationResult, error) {
			return &domain.VerificationResult{
				UUID:    uuid,
				Status:  domain.StatusInProcess,
				Message: "En proceso...",
			}, nil
		},
	}
	service := services.NewConciliatorService(mockGateway)

	// 2. Act
	result, err := service.VerifyRequest("RFC", "UUID", "c", "k")

	// 3. Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !contains(result, "El SAT sigue procesando tu solicitud") {
		t.Errorf("Expected message to indicate process, got: %s", result)
	}
}

func TestVerifyRequest_ErrorStatus(t *testing.T) {
	// 1. Arrange
	mockGateway := &MockSatGateway{
		CheckStatusFunc: func(rfc, uuid, c, k string) (*domain.VerificationResult, error) {
			return &domain.VerificationResult{
				UUID:    uuid,
				Status:  domain.StatusError,
				Message: "Error interno del SAT",
			}, nil
		},
	}
	service := services.NewConciliatorService(mockGateway)

	// 2. Act
	result, err := service.VerifyRequest("RFC", "UUID", "c", "k")

	// 3. Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !contains(result, "Estado: Error") {
		t.Errorf("Expected message to indicate error, got: %s", result)
	}
}

func TestDownloadPackage(t *testing.T) {
	// 1. Arrange
	expectedData := []byte("DATA")
	mockGateway := &MockSatGateway{
		DownloadPackageFunc: func(rfc, id, c, k string) ([]byte, error) {
			return expectedData, nil
		},
	}
	service := services.NewConciliatorService(mockGateway)

	// 2. Act
	data, err := service.DownloadPackage("RFC", "PKG-1", "c", "k")

	// 3. Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(data) != string(expectedData) {
		t.Errorf("Expected %s, got %s", expectedData, data)
	}
}

func TestCheckStatus(t *testing.T) {
	// 1. Arrange
	mockGateway := &MockSatGateway{
		CheckStatusFunc: func(rfc, uuid, c, k string) (*domain.VerificationResult, error) {
			return &domain.VerificationResult{Status: domain.StatusRejected}, nil
		},
	}
	service := services.NewConciliatorService(mockGateway)

	// 2. Act
	res, err := service.CheckStatus("RFC", "UUID", "c", "k")

	// 3. Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res.Status != domain.StatusRejected {
		t.Errorf("Expected Rejected, got %v", res.Status)
	}
}

// Helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 &&
		(s == substr || (len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
