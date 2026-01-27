package domain

type domainErr struct {
	message string
}

func (e domainErr) Error() string {
	return e.message
}

type NotFoundErr struct {
	domainErr
}

func NewNotFoundErr(message string) *NotFoundErr {
	return &NotFoundErr{
		domainErr: domainErr{message: message},
	}
}

type ValidationErr struct {
	domainErr
}

func NewValidationErr(message string) *ValidationErr {
	return &ValidationErr{
		domainErr: domainErr{message: message},
	}
}
