package main

import (
	"io"

	capn "github.com/glycerine/go-capnproto"
)

// concatenate cerealize/{packet.go saveload.go packet.capnp.go}

// PelicanPacket describes the packets exchanged between Chaser and LongPoller.
// We use the underlying (bambam generated) capnproto struct, PelicanPacketCapn,
// directly to avoid copying Payload around too often. This
// PelicanPacket struct acts as IDL, and debugging convenience.
//
// Either RequestSer must be -1 (for a Response) or ResponseSer must be -1 (for a request).
//
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

func (s *PelicanPacket) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	PelicanPacketGoToCapn(seg, s)
	_, err := seg.WriteTo(w)
	return err
}

func (s *PelicanPacket) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		//panic(fmt.Errorf("capn.ReadFromStream error: %s", err))
		return err
	}
	z := ReadRootPelicanPacketCapn(capMsg)
	PelicanPacketCapnToGo(z, s)
	return nil
}

func PelicanPacketCapnToGo(src PelicanPacketCapn, dest *PelicanPacket) *PelicanPacket {
	if dest == nil {
		dest = &PelicanPacket{}
	}
	dest.RequestSer = int64(src.RequestSer())
	dest.ResponseSer = int64(src.ResponseSer())
	dest.Paysize = int64(src.Paysize())
	dest.RequestAbTm = int64(src.RequestAbTm())
	dest.RequestLpTm = int64(src.RequestLpTm())
	dest.ResponseLpTm = int64(src.ResponseLpTm())
	dest.ResponseAbTm = int64(src.ResponseAbTm())
	dest.Key = src.Key()

	var n int

	// Paymac
	n = src.Paymac().Len()
	dest.Paymac = make([]byte, n)
	for i := 0; i < n; i++ {
		dest.Paymac[i] = byte(src.Paymac().At(i))
	}

	// Payload
	n = src.Payload().Len()
	dest.Payload = make([]byte, n)
	for i := 0; i < n; i++ {
		dest.Payload[i] = byte(src.Payload().At(i))
	}

	return dest
}

func PelicanPacketGoToCapn(seg *capn.Segment, src *PelicanPacket) PelicanPacketCapn {
	dest := AutoNewPelicanPacketCapn(seg)
	dest.SetRequestSer(src.RequestSer)
	dest.SetResponseSer(src.ResponseSer)
	dest.SetPaysize(src.Paysize)
	dest.SetRequestAbTm(src.RequestAbTm)
	dest.SetRequestLpTm(src.RequestLpTm)
	dest.SetResponseLpTm(src.ResponseLpTm)
	dest.SetResponseAbTm(src.ResponseAbTm)
	dest.SetKey(src.Key)

	mylist1 := seg.NewUInt8List(len(src.Paymac))
	for i := range src.Paymac {
		mylist1.Set(i, uint8(src.Paymac[i]))
	}
	dest.SetPaymac(mylist1)

	mylist2 := seg.NewUInt8List(len(src.Payload))
	for i := range src.Payload {
		mylist2.Set(i, uint8(src.Payload[i]))
	}
	dest.SetPayload(mylist2)

	return dest
}

func SliceByteToUInt8List(seg *capn.Segment, m []byte) capn.UInt8List {
	lst := seg.NewUInt8List(len(m))
	for i := range m {
		lst.Set(i, uint8(m[i]))
	}
	return lst
}

func UInt8ListToSliceByte(p capn.UInt8List) []byte {
	v := make([]byte, p.Len())
	for i := range v {
		v[i] = byte(p.At(i))
	}
	return v
}

type PelicanPacketCapn capn.Struct

func NewPelicanPacketCapn(s *capn.Segment) PelicanPacketCapn {
	return PelicanPacketCapn(s.NewStruct(56, 3))
}
func NewRootPelicanPacketCapn(s *capn.Segment) PelicanPacketCapn {
	return PelicanPacketCapn(s.NewRootStruct(56, 3))
}
func AutoNewPelicanPacketCapn(s *capn.Segment) PelicanPacketCapn {
	return PelicanPacketCapn(s.NewStructAR(56, 3))
}
func ReadRootPelicanPacketCapn(s *capn.Segment) PelicanPacketCapn {
	return PelicanPacketCapn(s.Root(0).ToStruct())
}
func (s PelicanPacketCapn) RequestSer() int64          { return int64(capn.Struct(s).Get64(0)) }
func (s PelicanPacketCapn) SetRequestSer(v int64)      { capn.Struct(s).Set64(0, uint64(v)) }
func (s PelicanPacketCapn) ResponseSer() int64         { return int64(capn.Struct(s).Get64(8)) }
func (s PelicanPacketCapn) SetResponseSer(v int64)     { capn.Struct(s).Set64(8, uint64(v)) }
func (s PelicanPacketCapn) Paysize() int64             { return int64(capn.Struct(s).Get64(16)) }
func (s PelicanPacketCapn) SetPaysize(v int64)         { capn.Struct(s).Set64(16, uint64(v)) }
func (s PelicanPacketCapn) RequestAbTm() int64         { return int64(capn.Struct(s).Get64(24)) }
func (s PelicanPacketCapn) SetRequestAbTm(v int64)     { capn.Struct(s).Set64(24, uint64(v)) }
func (s PelicanPacketCapn) RequestLpTm() int64         { return int64(capn.Struct(s).Get64(32)) }
func (s PelicanPacketCapn) SetRequestLpTm(v int64)     { capn.Struct(s).Set64(32, uint64(v)) }
func (s PelicanPacketCapn) ResponseLpTm() int64        { return int64(capn.Struct(s).Get64(40)) }
func (s PelicanPacketCapn) SetResponseLpTm(v int64)    { capn.Struct(s).Set64(40, uint64(v)) }
func (s PelicanPacketCapn) ResponseAbTm() int64        { return int64(capn.Struct(s).Get64(48)) }
func (s PelicanPacketCapn) SetResponseAbTm(v int64)    { capn.Struct(s).Set64(48, uint64(v)) }
func (s PelicanPacketCapn) Key() string                { return capn.Struct(s).GetObject(0).ToText() }
func (s PelicanPacketCapn) SetKey(v string)            { capn.Struct(s).SetObject(0, s.Segment.NewText(v)) }
func (s PelicanPacketCapn) Paymac() capn.UInt8List     { return capn.UInt8List(capn.Struct(s).GetObject(1)) }
func (s PelicanPacketCapn) SetPaymac(v capn.UInt8List) { capn.Struct(s).SetObject(1, capn.Object(v)) }
func (s PelicanPacketCapn) Payload() capn.UInt8List {
	return capn.UInt8List(capn.Struct(s).GetObject(2))
}
func (s PelicanPacketCapn) SetPayload(v capn.UInt8List) { capn.Struct(s).SetObject(2, capn.Object(v)) }

type PelicanPacketCapn_List capn.PointerList

func NewPelicanPacketCapnList(s *capn.Segment, sz int) PelicanPacketCapn_List {
	return PelicanPacketCapn_List(s.NewCompositeList(56, 3, sz))
}
func (s PelicanPacketCapn_List) Len() int { return capn.PointerList(s).Len() }
func (s PelicanPacketCapn_List) At(i int) PelicanPacketCapn {
	return PelicanPacketCapn(capn.PointerList(s).At(i).ToStruct())
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
	capn.PointerList(s).Set(i, capn.Object(item))
}
