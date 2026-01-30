package domain

// errors.go defines domain-specific error types.
type domainErr struct {
	message string
}

// Error returns the error message.
func (e domainErr) Error() string {
	return e.message
}

// NotFoundErr represents an error when a requested entity is not found.
type NotFoundErr struct {
	domainErr
}

// NewNotFoundErr creates a new NotFoundErr with the given message.
func NewNotFoundErr(message string) *NotFoundErr {
	return &NotFoundErr{
		domainErr: domainErr{message: message},
	}
}

// ValidationErr represents an error when validation fails.
type ValidationErr struct {
	domainErr
}

// NewValidationErr creates a new ValidationErr with the given message.
func NewValidationErr(message string) *ValidationErr {
	return &ValidationErr{
		domainErr: domainErr{message: message},
	}
}
