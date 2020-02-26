package kvstore

import "fmt"

// Validation status codes
const (
	codeSuccess uint32 = iota
	codeInvalidFormat
	codeDatabaseErr
	codeDupValueErr
	codeHashErr
)

// Return a log message for a status code
func logForCode(code uint32) string {
	switch code {
	case codeSuccess:
		return "success"
	case codeInvalidFormat:
		return "error:invalidFormat"
	case codeDatabaseErr:
		return "error:database"
	case codeDupValueErr:
		return "error:dupValue"
	case codeHashErr:
		return "error:hash"
	default:
		return fmt.Sprintf("unknown code %d", code)
	}
}
