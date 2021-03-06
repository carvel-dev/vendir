package pivnet

import (
	"fmt"
	"net/http"
	"strings"
)

type pivnetErr struct {
	Message string   `json:"message"`
	Errors  []string `json:"errors"`
}

type pivnetInternalServerErr struct {
	Error string `json:"error"`
}

type ErrPivnetOther struct {
	ResponseCode int      `json:"response_code" yaml:"response_code"`
	Message      string   `json:"message" yaml:"message"`
	Errors       []string `json:"errors" yaml:"errors"`
}

func (e ErrPivnetOther) Error() string {
	return fmt.Sprintf(
		"%d - %s. Errors: %v",
		e.ResponseCode,
		e.Message,
		strings.Join(e.Errors, ","),
	)
}

type ErrUnauthorized struct {
	ResponseCode int    `json:"response_code" yaml:"response_code"`
	Message      string `json:"message" yaml:"message"`
}

func (e ErrUnauthorized) Error() string {
	return e.Message
}

func newErrUnauthorized(message string) ErrUnauthorized {
	return ErrUnauthorized{
		ResponseCode: http.StatusUnauthorized,
		Message:      message,
	}
}

type ErrNotFound struct {
	ResponseCode int    `json:"response_code" yaml:"response_code"`
	Message      string `json:"message" yaml:"message"`
}

func (e ErrNotFound) Error() string {
	return e.Message
}

func newErrNotFound(message string) ErrNotFound {
	return ErrNotFound{
		ResponseCode: http.StatusNotFound,
		Message:      message,
	}
}

type ErrUnavailableForLegalReasons struct {
	ResponseCode int    `json:"response_code" yaml:"response_code"`
	Message      string `json:"message" yaml:"message"`
}

func (e ErrUnavailableForLegalReasons) Error() string {
	return e.Message
}

func newErrUnavailableForLegalReasons(message string) ErrUnavailableForLegalReasons {
	return ErrUnavailableForLegalReasons{
		ResponseCode: http.StatusUnavailableForLegalReasons,
		Message:      message,
	}
}

type ErrTooManyRequests struct {
	ResponseCode int    `json:"response_code" yaml:"response_code"`
	Message      string `json:"message" yaml:"message"`
}

func (e ErrTooManyRequests) Error() string {
	return e.Message
}

func newErrTooManyRequests() ErrTooManyRequests {
	return ErrTooManyRequests{
		ResponseCode: http.StatusTooManyRequests,
		Message: "You have hit a rate limit for this request",
	}
}