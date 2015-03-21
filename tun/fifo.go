package pelicantun

// request queue can be empty
type RequestFifo struct {
	q    []*tunnelPacket
	size int
}

func NewRequestFifo(capacity int) *RequestFifo {
	r := &RequestFifo{
		q:    make([]*tunnelPacket, 0, capacity),
		size: capacity,
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

func (s *RequestFifo) PushLeft(by *tunnelPacket) {
	s.q = append([]*tunnelPacket{by}, s.q...)
}

func (s *RequestFifo) PushRight(by *tunnelPacket) {
	s.q = append(s.q, by)
}

func (s *RequestFifo) PopRight() *tunnelPacket {
	r := s.PeekRight()
	n := len(s.q)
	s.q = s.q[:n-1]
	return r
}

func (s *RequestFifo) PeekRight() *tunnelPacket {
	if len(s.q) == 0 {
		return nil
	}
	n := len(s.q)
	r := s.q[n-1]
	return r
}
