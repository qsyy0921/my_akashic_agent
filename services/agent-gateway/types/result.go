package types

type ErrorCode string

const (
	ErrorCodeOK              ErrorCode = "OK"
	ErrorCodeInvalidArgument ErrorCode = "INVALID_ARGUMENT"
	ErrorCodeInternal        ErrorCode = "INTERNAL"
)

type Result struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message,omitempty"`
	Data    any       `json:"data,omitempty"`
}
