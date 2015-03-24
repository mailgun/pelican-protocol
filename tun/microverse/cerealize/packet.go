package cerealize

// the packets exchanged between Chaser and LongPoller.
// We use the underlying (bambam generated) capnproto struct directly to avoid
// copying Payload around too often.
type PelicanPacket struct {
	RequestSer   int64  `capid:"0"` // -1 if this is a response
	ResponseSer  int64  `capid:"1"` // -1 if this is a request
	Paysize      int64  `capid:"2"`
	RequestAbTm  int64  `capid:"3"`
	RequestLpTm  int64  `capid:"4"`
	ResponseLpTm int64  `capid:"5"`
	ResponseAbTm int64  `capid:"6"`
	Key          string `capid:"7"`
	Paymac       []byte `capid:"8"`
	Payload      []byte `capid:"9"`
}
