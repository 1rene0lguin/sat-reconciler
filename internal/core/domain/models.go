package domain

type RequestStatus int

const (
	StatusAccepted  RequestStatus = 1
	StatusInProcess RequestStatus = 2
	StatusFinished  RequestStatus = 3
	StatusError     RequestStatus = 4
	StatusRejected  RequestStatus = 5
)

func (r RequestStatus) String() string {
	switch r {
	case StatusAccepted:
		return "Aceptada"
	case StatusInProcess:
		return "En proceso"
	case StatusFinished:
		return "Terminada"
	case StatusError:
		return "Error"
	case StatusRejected:
		return "Rechazada"
	default:
		return "Desconocido"
	}
}

type VerificationResult struct {
	UUID       string
	Status     RequestStatus
	Message    string
	PackageIDs []string
}
