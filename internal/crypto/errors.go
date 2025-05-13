package crypto

type ErrorIdType int

const (
	NoError ErrorIdType = iota + 1
        LibraryError
        AsyncOperation
)

type CryptoError struct {
	StatusCode ErrorIdType
	Err error
}

func ErrorCode(err any) ErrorIdType {
	if err == nil {
		return NoError
	}
	re, ok := err.(*CryptoError)
	if ok {
		return re.StatusCode
	}
	return LibraryError
}
