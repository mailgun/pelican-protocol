package schema

// the packets exchanged between Chaser and LongPoller.
// We use the underlying (bambam generated) capnproto struct directly to avoid
// copying Payload around too often.
type PelicanPacket struct {
	ResponseSerial int64  `capid:"0"` // -1 if this is a request
	RequestSerial  int64  `capid:"1"` // -1 if this is a response
	Key            string `capid:"2"`
	Mac            []byte `capid:"3"`
	Payload        []byte `capid:"4"`
	RequestAbTm    int64  `capid:"5"`
	RequestLpTm    int64  `capid:"6"`
	ResponseLpTm   int64  `capid:"7"`
	ResponseAbTm   int64  `capid:"8"`
}
