package domain

type RequestStatus int

const (
	StatusAccepted  RequestStatus = 1
	StatusInProcess RequestStatus = 2
	StatusFinished  RequestStatus = 3
	StatusError     RequestStatus = 4
	StatusRejected  RequestStatus = 5
)

type VerificationResult struct {
	UUID       string
	Status     RequestStatus
	Message    string
	PackageIDs []string
}
