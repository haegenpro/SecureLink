package worker

// PermanentError marks a processing failure as non-retryable: the file is
// recorded as domain.StatusFailed and the SQS message is acknowledged
// (deleted) rather than left for redelivery, per Spec.md section 6
// ("Permanent processing failure -> mark the file as failed").
//
// Any other error returned from a Processor is treated as transient: the
// message is left un-deleted so SQS redelivers it after the visibility
// timeout.
type PermanentError struct {
	Err error
}

func Permanent(err error) error {
	return &PermanentError{Err: err}
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}
