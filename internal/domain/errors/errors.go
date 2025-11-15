package errors

import "errors"

var (
	ErrTeamExists   = errors.New("TEAM_EXISTS")
	ErrPRExists     = errors.New("PR_EXISTS")
	ErrPRMerged     = errors.New("PR_MERGED")
	ErrNotAssigned  = errors.New("NOT_ASSIGNED")
	ErrNoCandidate  = errors.New("NO_CANDIDATE")
	ErrNotFound     = errors.New("NOT_FOUND")
	ErrUnauthorized = errors.New("UNAUTHORIZED")
	ErrInvalidInput = errors.New("INVALID_INPUT")
)

// DomainError представляет доменную ошибку с кодом и сообщением
type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError создает новую доменную ошибку
func NewDomainError(code, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
