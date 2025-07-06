package errors

import (
	"errors"
	"fmt"
)

var (
	ErrSlotNotFound       = errors.New("slot not found")
	ErrFutureSlot         = errors.New("requested slot is in the future")
	ErrSlotTooFarInFuture = errors.New("requested slot is too far in the future")
	ErrInvalidSlot        = errors.New("invalid slot number")
	ErrRPCConnection      = errors.New("RPC connection error")
	ErrTimeout            = errors.New("request timeout")
	ErrInternal           = errors.New("internal server error")
)

type ValidationError struct {
	Field string
	Value interface{}
	Err   error
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s with value %v: %v", e.Field, e.Value, e.Err)
}

func (e ValidationError) Unwrap() error {
	return e.Err
}

type RPCError struct {
	Code    int
	Message string
	Data    interface{}
}

func (e RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

func NewValidationError(field string, value interface{}, err error) error {
	return ValidationError{
		Field: field,
		Value: value,
		Err:   err,
	}
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrSlotNotFound)
}

func IsBadRequest(err error) bool {
	return errors.Is(err, ErrFutureSlot) ||
		errors.Is(err, ErrInvalidSlot) ||
		errors.Is(err, ErrSlotTooFarInFuture)
}

func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}
