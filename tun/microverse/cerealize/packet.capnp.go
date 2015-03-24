package cerealize

// AUTO GENERATED - DO NOT EDIT

import (
	C "github.com/glycerine/go-capnproto"
)

type PelicanPacketCapn C.Struct

func NewPelicanPacketCapn(s *C.Segment) PelicanPacketCapn {
	return PelicanPacketCapn(s.NewStruct(56, 3))
}
func NewRootPelicanPacketCapn(s *C.Segment) PelicanPacketCapn {
	return PelicanPacketCapn(s.NewRootStruct(56, 3))
}
func AutoNewPelicanPacketCapn(s *C.Segment) PelicanPacketCapn {
	return PelicanPacketCapn(s.NewStructAR(56, 3))
}
func ReadRootPelicanPacketCapn(s *C.Segment) PelicanPacketCapn {
	return PelicanPacketCapn(s.Root(0).ToStruct())
}
func (s PelicanPacketCapn) RequestSer() int64        { return int64(C.Struct(s).Get64(0)) }
func (s PelicanPacketCapn) SetRequestSer(v int64)    { C.Struct(s).Set64(0, uint64(v)) }
func (s PelicanPacketCapn) ResponseSer() int64       { return int64(C.Struct(s).Get64(8)) }
func (s PelicanPacketCapn) SetResponseSer(v int64)   { C.Struct(s).Set64(8, uint64(v)) }
func (s PelicanPacketCapn) Paysize() int64           { return int64(C.Struct(s).Get64(16)) }
func (s PelicanPacketCapn) SetPaysize(v int64)       { C.Struct(s).Set64(16, uint64(v)) }
func (s PelicanPacketCapn) RequestAbTm() int64       { return int64(C.Struct(s).Get64(24)) }
func (s PelicanPacketCapn) SetRequestAbTm(v int64)   { C.Struct(s).Set64(24, uint64(v)) }
func (s PelicanPacketCapn) RequestLpTm() int64       { return int64(C.Struct(s).Get64(32)) }
func (s PelicanPacketCapn) SetRequestLpTm(v int64)   { C.Struct(s).Set64(32, uint64(v)) }
func (s PelicanPacketCapn) ResponseLpTm() int64      { return int64(C.Struct(s).Get64(40)) }
func (s PelicanPacketCapn) SetResponseLpTm(v int64)  { C.Struct(s).Set64(40, uint64(v)) }
func (s PelicanPacketCapn) ResponseAbTm() int64      { return int64(C.Struct(s).Get64(48)) }
func (s PelicanPacketCapn) SetResponseAbTm(v int64)  { C.Struct(s).Set64(48, uint64(v)) }
func (s PelicanPacketCapn) Key() string              { return C.Struct(s).GetObject(0).ToText() }
func (s PelicanPacketCapn) SetKey(v string)          { C.Struct(s).SetObject(0, s.Segment.NewText(v)) }
func (s PelicanPacketCapn) Paymac() C.UInt8List      { return C.UInt8List(C.Struct(s).GetObject(1)) }
func (s PelicanPacketCapn) SetPaymac(v C.UInt8List)  { C.Struct(s).SetObject(1, C.Object(v)) }
func (s PelicanPacketCapn) Payload() C.UInt8List     { return C.UInt8List(C.Struct(s).GetObject(2)) }
func (s PelicanPacketCapn) SetPayload(v C.UInt8List) { C.Struct(s).SetObject(2, C.Object(v)) }

// capn.JSON_enabled == false so we stub MarshallJSON().
func (s PelicanPacketCapn) MarshalJSON() (bs []byte, err error) { return }

type PelicanPacketCapn_List C.PointerList

func NewPelicanPacketCapnList(s *C.Segment, sz int) PelicanPacketCapn_List {
	return PelicanPacketCapn_List(s.NewCompositeList(56, 3, sz))
}
func (s PelicanPacketCapn_List) Len() int { return C.PointerList(s).Len() }
func (s PelicanPacketCapn_List) At(i int) PelicanPacketCapn {
	return PelicanPacketCapn(C.PointerList(s).At(i).ToStruct())
}
func (s PelicanPacketCapn_List) ToArray() []PelicanPacketCapn {
	n := s.Len()
	a := make([]PelicanPacketCapn, n)
	for i := 0; i < n; i++ {
		a[i] = s.At(i)
	}
	return a
}
func (s PelicanPacketCapn_List) Set(i int, item PelicanPacketCapn) {
	C.PointerList(s).Set(i, C.Object(item))
}
