package thriftbp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/avast/retry-go"

	"github.com/reddit/baseplate.go/errorsbp"
	baseplatethrift "github.com/reddit/baseplate.go/internal/gen-go/reddit/baseplate"
	"github.com/reddit/baseplate.go/retrybp"
)

// baseplateError defines the interface of thrift compiled baseplate.Error
// that will be satisfied as long as services are using the same version of
// thrift compiler.
type baseplateError interface {
	thrift.TException

	IsSetMessage() bool
	GetMessage() string

	IsSetCode() bool
	GetCode() int32

	IsSetRetryable() bool
	GetRetryable() bool

	IsSetDetails() bool
	GetDetails() map[string]string
}

var (
	_ baseplateError = (*baseplatethrift.Error)(nil)
)

// ClientPoolConfig errors are returned if the configuration validation fails.
var (
	ErrConfigMissingServiceSlug = errors.New("`ServiceSlug` cannot be empty")
	ErrConfigMissingAddr        = errors.New("`Addr` cannot be empty")
	ErrConfigInvalidConnections = errors.New("`InitialConnections` cannot be bigger than `MaxConnections`")
)

// WithDefaultRetryableCodes returns a list including the given error codes and
// the default retryable error codes:
//
// 1. TOO_EARLY
//
// 2. TOO_MANY_REQUESTS
//
// 3. SERVICE_UNAVAILABLE
func WithDefaultRetryableCodes(codes ...int32) []int32 {
	return append([]int32{
		int32(baseplatethrift.ErrorCode_TOO_EARLY),
		int32(baseplatethrift.ErrorCode_TOO_MANY_REQUESTS),
		int32(baseplatethrift.ErrorCode_SERVICE_UNAVAILABLE),
	}, codes...)
}

// BaseplateErrorFilter returns true if the given error is a baseplate.Error
// and returns one of the given codes and false if it is a baseplate.Error
// but does not return one of the given codes otherwise it calls the next filter
// in the chain.
func BaseplateErrorFilter(codes ...int32) retrybp.Filter {
	codeMap := make(map[int32]bool, len(codes))
	for _, code := range codes {
		codeMap[code] = true
	}
	return func(err error, next retry.RetryIfFunc) bool {
		var bpErr baseplateError
		if errors.As(err, &bpErr) {
			return codeMap[bpErr.GetCode()]
		}
		return next(err)
	}
}

// IDLExceptionSuppressor is an errorsbp.Suppressor implementation that returns
// true on errors from exceptions defined in thrift IDL files.
//
// Note that if the exception is baseplate.Error,
// this function will NOT suppress it if the code is in range [500, 600).
func IDLExceptionSuppressor(err error) bool {
	var te thrift.TException
	if !errors.As(err, &te) {
		return false
	}
	var bpErr baseplateError
	if errors.As(err, &bpErr) {
		// If this is also baseplate.Error,
		// only suppress it if the error code is outside of [500, 600).
		code := bpErr.GetCode()
		return code < 500 || code >= 600
	}
	return te.TExceptionType() == thrift.TExceptionTypeCompiled
}

var _ errorsbp.Suppressor = IDLExceptionSuppressor

type wrappedBaseplateError struct {
	cause error
	bpErr baseplateError
}

func (e wrappedBaseplateError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("baseplate.Error: %q", e.bpErr.GetMessage()))
	first := true
	writeSeparator := func() {
		if first {
			first = false
			sb.WriteString(" (")
		} else {
			sb.WriteString(", ")
		}
	}
	if e.bpErr.IsSetCode() {
		writeSeparator()
		sb.WriteString(fmt.Sprintf("code=%d", e.bpErr.GetCode()))
	}
	if e.bpErr.IsSetRetryable() {
		writeSeparator()
		sb.WriteString(fmt.Sprintf("retryable=%v", e.bpErr.GetRetryable()))
	}
	if e.bpErr.IsSetDetails() {
		writeSeparator()
		sb.WriteString(fmt.Sprintf("details=%#v", e.bpErr.GetDetails()))
	}
	if !first {
		sb.WriteString(")")
	}
	return sb.String()
}

func (e wrappedBaseplateError) Unwrap() error {
	return e.cause
}

// WrapBaseplateError wraps e to an error with more meaningful error message if
// e is baseplate.Error. Otherwise it returns e as-is.
//
// NOTE: This in general should only be used in clients.
// If you wrap baseplate.Error returned in server code,
// it could cause the error no longer being recognized as an error defined in
// thrift IDL by the thrift server, and the client would get a generic
// TApplicationException instead.
func WrapBaseplateError(e error) error {
	if errors.As(e, new(wrappedBaseplateError)) {
		// Already wrapped, return e as-is to avoid double wrapping
		return e
	}

	var bpErr baseplateError
	if errors.As(e, &bpErr) {
		return wrappedBaseplateError{
			cause: e,
			bpErr: bpErr,
		}
	}
	return e
}
