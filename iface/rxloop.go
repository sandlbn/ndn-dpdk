package iface

/*
#include "rxloop.h"

uint16_t go_ChanRxGroup_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"errors"
	"sync"
	"sync/atomic"
	"unsafe"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

// Receive channel for a group of faces.
type IRxGroup interface {
	GetPtr() unsafe.Pointer
	getPtr() *C.RxGroup
	GetRxLoop() *RxLoop
	setRxLoop(rxl *RxLoop)

	GetNumaSocket() dpdk.NumaSocket
	ListFaces() []FaceId
}

// Base type to implement IRxGroup.
type RxGroupBase struct {
	c   unsafe.Pointer
	rxl *RxLoop
}

func (rxg *RxGroupBase) InitRxgBase(c unsafe.Pointer) {
	rxg.c = c
}

func (rxg *RxGroupBase) GetPtr() unsafe.Pointer {
	return rxg.c
}

func (rxg *RxGroupBase) getPtr() *C.RxGroup {
	return (*C.RxGroup)(rxg.c)
}

func (rxg *RxGroupBase) GetRxLoop() *RxLoop {
	return rxg.rxl
}

func (rxg *RxGroupBase) setRxLoop(rxl *RxLoop) {
	rxg.rxl = rxl
}

// An RxGroup using a Go channel as receive queue.
type ChanRxGroup struct {
	RxGroupBase
	nFaces int32    // accessed via atomic.AddInt32
	faces  sync.Map // map[FaceId]IFace
	queue  chan dpdk.Packet
}

func newChanRxGroup() (rxg *ChanRxGroup) {
	rxg = new(ChanRxGroup)
	C.theChanRxGroup_.rxBurstOp = C.RxGroup_RxBurst(C.go_ChanRxGroup_RxBurst)
	rxg.InitRxgBase(unsafe.Pointer(&C.theChanRxGroup_))
	rxg.queue = make(chan dpdk.Packet, 1024)
	return rxg
}

// Change queue capacity (not thread safe).
func (rxg *ChanRxGroup) SetQueueCapacity(queueCapacity int) {
	rxg.queue = make(chan dpdk.Packet, queueCapacity)
}

func (rxg *ChanRxGroup) GetNumaSocket() dpdk.NumaSocket {
	return dpdk.NUMA_SOCKET_ANY
}

func (rxg *ChanRxGroup) ListFaces() (list []FaceId) {
	rxg.faces.Range(func(faceId, face interface{}) bool {
		list = append(list, faceId.(FaceId))
		return true
	})
	return list
}

func (rxg *ChanRxGroup) AddFace(face IFace) {
	if atomic.AddInt32(&rxg.nFaces, 1) == 1 {
		EmitRxGroupAdd(rxg)
	}
	rxg.faces.Store(face.GetFaceId(), face)
}

func (rxg *ChanRxGroup) RemoveFace(face IFace) {
	rxg.faces.Delete(face.GetFaceId())
	if atomic.AddInt32(&rxg.nFaces, -1) == 0 {
		EmitRxGroupRemove(rxg)
	}
}

func (rxg *ChanRxGroup) Rx(pkt dpdk.Packet) {
	select {
	case rxg.queue <- pkt:
	default:
		// TODO count drops
		pkt.Close()
	}
}

//export go_ChanRxGroup_RxBurst
func go_ChanRxGroup_RxBurst(rxg *C.RxGroup, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	select {
	case pkt := <-TheChanRxGroup.queue:
		*pkts = (*C.struct_rte_mbuf)(pkt.GetPtr())
		return 1
	default:
	}
	return 0
}

var TheChanRxGroup = newChanRxGroup()

// LCoreAlloc role for RxLoop.
const LCoreRole_RxLoop = "RX"

// RX loop.
type RxLoop struct {
	dpdk.ThreadBase
	c          *C.RxLoop
	numaSocket dpdk.NumaSocket
	rxgs       map[*C.RxGroup]IRxGroup
}

func NewRxLoop(numaSocket dpdk.NumaSocket) (rxl *RxLoop) {
	rxl = new(RxLoop)
	rxl.ResetThreadBase()
	rxl.c = (*C.RxLoop)(dpdk.Zmalloc("RxLoop", C.sizeof_RxLoop, numaSocket))
	dpdk.InitStopFlag(unsafe.Pointer(&rxl.c.stop))
	rxl.numaSocket = numaSocket
	rxl.rxgs = make(map[*C.RxGroup]IRxGroup)
	return rxl
}

func (rxl *RxLoop) GetNumaSocket() dpdk.NumaSocket {
	return rxl.numaSocket
}

func (rxl *RxLoop) SetCallback(cb unsafe.Pointer, cbarg unsafe.Pointer) {
	rxl.c.cb = C.Face_RxCb(cb)
	rxl.c.cbarg = cbarg
}

func (rxl *RxLoop) Launch() error {
	return rxl.LaunchImpl(func() int {
		rs := urcu.NewReadSide()
		defer rs.Close()

		burst := NewRxBurst(64)
		defer burst.Close()
		rxl.c.burst = burst.c

		C.RxLoop_Run(rxl.c)
		return 0
	})
}

func (rxl *RxLoop) Stop() error {
	return rxl.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&rxl.c.stop)))
}

func (rxl *RxLoop) Close() error {
	if rxl.IsRunning() {
		return dpdk.ErrCloseRunningThread
	}

	for _, rxg := range rxl.rxgs {
		rxg.setRxLoop(nil)
	}

	dpdk.Free(rxl.c)
	return nil
}

func (rxl *RxLoop) ListRxGroups() (list []IRxGroup) {
	for _, rxg := range rxl.rxgs {
		list = append(list, rxg)
	}
	return list
}

func (rxl *RxLoop) ListFaces() (list []FaceId) {
	for _, rxg := range rxl.rxgs {
		list = append(list, rxg.ListFaces()...)
	}
	return list
}

func (rxl *RxLoop) AddRxGroup(rxg IRxGroup) error {
	if rxg.GetRxLoop() != nil {
		return errors.New("RxGroup is active in another RxLoop")
	}
	rxgC := rxg.getPtr()
	if rxgC.rxBurstOp == nil {
		return errors.New("RxGroup.rxBurstOp is missing")
	}

	rs := urcu.NewReadSide()
	defer rs.Close()

	if rxl.numaSocket == dpdk.NUMA_SOCKET_ANY {
		rxl.numaSocket = rxg.GetNumaSocket()
	}
	rxl.rxgs[rxgC] = rxg

	rxg.setRxLoop(rxl)
	C.cds_hlist_add_head_rcu(&rxgC.rxlNode, &rxl.c.head)
	return nil
}

func (rxl *RxLoop) RemoveRxGroup(rxg IRxGroup) error {
	rs := urcu.NewReadSide()
	defer rs.Close()

	rxgC := rxg.getPtr()
	C.cds_hlist_del_rcu(&rxgC.rxlNode)
	urcu.Barrier()

	rxg.setRxLoop(nil)
	delete(rxl.rxgs, rxgC)
	return nil
}
