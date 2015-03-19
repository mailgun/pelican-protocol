package main

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		header: make(http.Header),
	}
}

type MockResponseWriter struct {
	store   bytes.Buffer
	header  http.Header
	errcode int
}

//type ResponseWriter interface methods

// Header returns the header map that will be sent by WriteHeader.
// Changing the header after a call to WriteHeader (or Write) has
// no effect.
func (s *MockResponseWriter) Header() http.Header {
	return s.header
}

// Write writes the data to the connection as part of an HTTP reply.
// If WriteHeader has not yet been called, Write calls WriteHeader(http.StatusOK)
// before writing the data.  If the Header does not contain a
// Content-Type line, Write adds a Content-Type set to the result of passing
// the initial 512 bytes of written data to DetectContentType.
func (s *MockResponseWriter) Write(p []byte) (int, error) {
	return s.store.Write(p)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (s *MockResponseWriter) WriteHeader(errcode int) {
	s.errcode = errcode
}
