package main

// request queue can be empty
type RequestFifo struct {
	q    [][]byte
	size int
}

func NewRequestFifo(size int) *RequestFifo {
	r := &RequestFifo{
		q:    make([][]byte, 0, size),
		size: size,
	}
	return r
}

func (s *RequestFifo) Len() int {
	return len(s.q)
}

func (s *RequestFifo) Empty() bool {
	if len(s.q) == 0 {
		return true
	}
	return false
}

func (s *RequestFifo) PushLeft(by []byte) {
	s.q = append([][]byte{by}, s.q...)
}
func (s *RequestFifo) PushRight(by []byte) {
	s.q = append(s.q, by)
}

func (s *RequestFifo) PopRight() []byte {
	r := s.PeekRight()
	n := len(s.q)
	s.q = s.q[:n-1]
	return r
}

func (s *RequestFifo) PeekRight() []byte {
	if len(s.q) == 0 {
		return nil
	}
	n := len(s.q)
	r := s.q[n-1]
	return r
}
