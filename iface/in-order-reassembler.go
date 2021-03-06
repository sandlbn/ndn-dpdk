package iface

/*
#include "in-order-reassembler.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/ndn"
)

type InOrderReassembler struct {
	c *C.InOrderReassembler
}

func NewInOrderReassembler() InOrderReassembler {
	return InOrderReassembler{new(C.InOrderReassembler)}
}

func InOrderReassemblerFromPtr(ptr unsafe.Pointer) InOrderReassembler {
	return InOrderReassembler{(*C.InOrderReassembler)(ptr)}
}

func (r InOrderReassembler) Receive(pkt ndn.Packet) ndn.Packet {
	res := C.InOrderReassembler_Receive(r.c, (*C.Packet)(pkt.GetPtr()))
	return ndn.PacketFromPtr(unsafe.Pointer(res))
}

type InOrderReassemblerCounters struct {
	Accepted   uint64
	OutOfOrder uint64
	Delivered  uint64
	Incomplete uint64
}

func (cnt InOrderReassemblerCounters) String() string {
	return fmt.Sprintf("%dacpt %dooo %ddlvr %dincomp",
		cnt.Accepted, cnt.OutOfOrder, cnt.Delivered, cnt.Incomplete)
}

func (r InOrderReassembler) ReadCounters() InOrderReassemblerCounters {
	return InOrderReassemblerCounters{
		Accepted:   uint64(r.c.nAccepted),
		OutOfOrder: uint64(r.c.nOutOfOrder),
		Delivered:  uint64(r.c.nDelivered),
		Incomplete: uint64(r.c.nIncomplete),
	}
}
