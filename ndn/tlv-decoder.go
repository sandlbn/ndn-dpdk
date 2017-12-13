package ndn

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../build-c -lndn-traffic-dpdk-dpdk

#include "tlv-decoder.h"
*/
import "C"
import (
	"ndn-traffic-dpdk/dpdk"
)

type TlvDecoder struct {
	it dpdk.PacketIterator
}

func NewTlvDecoder(pkt dpdk.Packet) TlvDecoder {
	return TlvDecoder{dpdk.NewPacketIterator(pkt)}
}

func (d *TlvDecoder) getPtr() *C.TlvDecoder {
	return (*C.TlvDecoder)(d.it.GetPtr())
}

// Decode a TLV-TYPE or TLV-LENGTH number.
func (d *TlvDecoder) ReadVarNum() (v uint64, length uint, e error) {
	var lengthC C.size_t
	res := C.DecodeVarNum(d.getPtr(), (*C.uint64_t)(&v), &lengthC)
	if res != C.NdnError_OK {
		return 0, 0, NdnError(res)
	}
	return v, uint(lengthC), nil
}