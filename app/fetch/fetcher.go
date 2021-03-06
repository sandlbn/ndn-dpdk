package fetch

/*
#include "fetcher.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/pktqueue"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type FetcherConfig struct {
	NThreads       int
	NProcs         int
	RxQueue        pktqueue.Config
	WindowCapacity int
}

// Fetcher controls fetch threads and fetch procedures on a face.
type Fetcher struct {
	fth          []*fetchThread
	fp           []*C.FetchProc
	nActiveProcs int
}

func New(face iface.IFace, cfg FetcherConfig) (fetcher *Fetcher, e error) {
	if cfg.NThreads == 0 {
		cfg.NThreads = 1
	}
	if cfg.NProcs == 0 {
		cfg.NProcs = 1
	}
	cfg.RxQueue.DisableCoDel = true

	faceId := face.GetFaceId()
	socket := face.GetNumaSocket()
	interestMp := (*C.struct_rte_mempool)(appinit.MakePktmbufPool(appinit.MP_INT, socket).GetPtr())

	fetcher = new(Fetcher)
	fetcher.fth = make([]*fetchThread, cfg.NThreads)
	for i := range fetcher.fth {
		fth := new(fetchThread)
		fth.c = (*C.FetchThread)(dpdk.Zmalloc("FetchThread", C.sizeof_FetchThread, socket))
		fth.c.face = (C.FaceId)(faceId)
		fth.c.interestMp = interestMp
		C.NonceGen_Init(&fth.c.nonceGen)
		fth.ResetThreadBase()
		dpdk.InitStopFlag(unsafe.Pointer(&fth.c.stop))
		fetcher.fth[i] = fth
	}

	fetcher.fp = make([]*C.FetchProc, cfg.NProcs)
	for i := range fetcher.fp {
		fp := (*C.FetchProc)(dpdk.Zmalloc("FetchProc", C.sizeof_FetchProc, socket))
		if _, e := pktqueue.NewAt(unsafe.Pointer(&fp.rxQueue), cfg.RxQueue, fmt.Sprintf("Fetcher%d-%d_rxQ", faceId, i), socket); e != nil {
			return nil, e
		}
		fp.pitToken = (C.uint64_t(i) << 56) | 0x6665746368 // 'fetch'
		fetcher.fp[i] = fp
		fetcher.GetLogic(i).Init(cfg.WindowCapacity, socket)
	}

	return fetcher, nil
}

func (fetcher *Fetcher) GetFace() iface.IFace {
	return iface.Get(iface.FaceId(fetcher.fth[0].c.face))
}

func (fetcher *Fetcher) CountThreads() int {
	return len(fetcher.fth)
}

func (fetcher *Fetcher) GetThread(i int) dpdk.IThread {
	return fetcher.fth[i]
}

func (fetcher *Fetcher) CountProcs() int {
	return len(fetcher.fp)
}

func (fetcher *Fetcher) GetRxQueue(i int) pktqueue.PktQueue {
	return pktqueue.FromPtr(unsafe.Pointer(&fetcher.fp[i].rxQueue))
}

func (fetcher *Fetcher) GetLogic(i int) *Logic {
	return LogicFromPtr(unsafe.Pointer(&fetcher.fp[i].logic))
}

func (fetcher *Fetcher) Reset() {
	for _, fth := range fetcher.fth {
		fth.c.head.next = nil
	}
	for i := range fetcher.fp {
		fetcher.GetLogic(i).Reset()
	}
	fetcher.nActiveProcs = 0
}

// Set name prefix and other InterestTemplate arguments.
func (fetcher *Fetcher) AddTemplate(tplArgs ...interface{}) (i int, e error) {
	i = fetcher.nActiveProcs
	if i >= len(fetcher.fp) {
		return -1, errors.New("too many prefixes")
	}

	fp := fetcher.fp[i]
	tpl := ndn.InterestTemplateFromPtr(unsafe.Pointer(&fp.tpl))
	if e := tpl.Init(append([]interface{}{ndn.InterestMbufExtraHeadroom(appinit.SizeofEthLpHeaders())}, tplArgs...)...); e != nil {
		return -1, e
	}

	if uintptr(tpl.PrefixL+1) >= unsafe.Sizeof(tpl.PrefixV) {
		return -1, errors.New("name too long")
	}
	tpl.PrefixV[tpl.PrefixL] = uint8(ndn.TT_SegmentNameComponent)
	// put SegmentNameComponent TLV-TYPE in the buffer so that it's checked in same memcmp

	rs := urcu.NewReadSide()
	defer rs.Close()
	fth := fetcher.fth[i%len(fetcher.fth)]
	C.cds_hlist_add_head_rcu(&fp.fthNode, &fth.c.head)
	fetcher.nActiveProcs++
	return i, nil
}

func (fetcher *Fetcher) Launch() {
	for _, fth := range fetcher.fth {
		fth.Launch()
	}
}

func (fetcher *Fetcher) Stop() {
	for _, fth := range fetcher.fth {
		fth.Stop()
	}
}

func (fetcher *Fetcher) Close() error {
	for i, fp := range fetcher.fp {
		fetcher.GetRxQueue(i).Close()
		fetcher.GetLogic(i).Close()
		dpdk.Free(fp)
	}
	for _, fth := range fetcher.fth {
		fth.Close()
	}
	return nil
}

type fetchThread struct {
	dpdk.ThreadBase
	c *C.FetchThread
}

func (fth *fetchThread) Launch() error {
	return fth.LaunchImpl(func() int {
		return int(C.FetchThread_Run(fth.c))
	})
}

func (fth *fetchThread) Stop() error {
	return fth.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&fth.c.stop)))
}

func (fth *fetchThread) Close() error {
	dpdk.Free(fth.c)
	return nil
}
