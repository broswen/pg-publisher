package publisher

type ErrUnknown struct {
	Err error
}

func (e ErrUnknown) Error() string {
	return e.Err.Error()
}

func (e ErrUnknown) Unwrap() error {
	return e.Err
}

type ErrNotFound struct {
	Err error
}

func (e ErrNotFound) Error() string {
	return e.Err.Error()
}

func (e ErrNotFound) Unwrap() error {
	return e.Err
}
