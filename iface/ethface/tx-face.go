package ethface

/*
#include "tx-face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func SizeofHeaderMempoolDataRoom() uint16 {
	return uint16(C.EthTxFace_GetHeaderMempoolDataRoom())
}

type TxFace struct {
	c *C.EthTxFace
}

func NewTxFace(q dpdk.EthTxQueue, indirectMp dpdk.PktmbufPool,
	headerMp dpdk.PktmbufPool) (face TxFace, e error) {
	if !hasValidFaceId(q.GetPort()) {
		return face, fmt.Errorf("port number is too large")
	}

	face.c = (*C.EthTxFace)(C.calloc(1, C.sizeof_EthTxFace))
	face.c.port = C.uint16_t(q.GetPort())
	face.c.queue = C.uint16_t(q.GetQueue())
	face.c.indirectMp = (*C.struct_rte_mempool)(indirectMp.GetPtr())
	face.c.headerMp = (*C.struct_rte_mempool)(headerMp.GetPtr())

	ok := C.EthTxFace_Init(face.c)
	if !ok {
		return face, dpdk.GetErrno()
	}
	return face, nil
}

func (face TxFace) GetFaceId() iface.FaceId {
	return FaceIdFromEthDev(uint16(face.c.port))
}

func (face TxFace) Close() error {
	C.EthTxFace_Close(face.c)
	C.free(unsafe.Pointer(face.c))
	return nil
}

func (face TxFace) TxBurst(pkts []ndn.Packet) {
	if len(pkts) == 0 {
		return
	}
	C.EthTxFace_TxBurst(face.c, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
}

type TxFaceCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64

	NFrames uint64 // total L2 frames
	NOctets uint64

	NL3Bursts     uint64
	NL3OverLength uint64
	NAllocFails   uint64
	NL2Bursts     uint64
	NL2Incomplete uint64
}

func (face TxFace) GetCounters() (cnt TxFaceCounters) {
	cnt.NInterests = uint64(face.c.nPkts[ndn.NdnPktType_Interest])
	cnt.NData = uint64(face.c.nPkts[ndn.NdnPktType_Interest])
	cnt.NNacks = uint64(face.c.nPkts[ndn.NdnPktType_Interest])

	cnt.NFrames = uint64(face.c.nPkts[ndn.NdnPktType_None]) + cnt.NInterests + cnt.NData + cnt.NNacks
	cnt.NOctets = uint64(face.c.nOctets)

	cnt.NL3Bursts = uint64(face.c.nL3Bursts)
	cnt.NL3OverLength = uint64(face.c.nL3OverLength)
	cnt.NAllocFails = uint64(face.c.nAllocFails)
	cnt.NL2Bursts = uint64(face.c.nL2Bursts)
	cnt.NL2Incomplete = uint64(face.c.nL2Incomplete)

	return cnt
}

func (cnt TxFaceCounters) String() string {
	return fmt.Sprintf(
		"%dI %dD %dN %dfrm %db; L3 %dbursts %doverlen %dallocfail; L2 %dbursts, %dincomplete",
		cnt.NInterests, cnt.NData, cnt.NNacks, cnt.NFrames, cnt.NOctets,
		cnt.NL3Bursts, cnt.NL3OverLength, cnt.NAllocFails, cnt.NL2Bursts, cnt.NL2Incomplete)
}
