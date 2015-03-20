package pelicantun

// request queue can be empty
type RequestFifo struct {
	q    []byte
	size int
}

func NewRequestFifo(size int) *RequestFifo {
	r := &RequestFifo{
		q:    make([]byte, 0, size),
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
	s.q = append(by, s.q...)
}

func (s *RequestFifo) PopRight() []byte {
	if len(s.q) == 0 {
		return nil
	}
	r := s.q[0]
	s.q = s.q[1:]
	return r
}
