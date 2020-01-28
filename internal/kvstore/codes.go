package kvstore

// Validation status codes
const (
	codeSuccess uint32 = iota
	codeInvalidFormat
	codeDatabaseErr
	codeDupValueErr
	codeHashErr
)
