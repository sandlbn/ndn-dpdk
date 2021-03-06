package fwdp

import (
	"fmt"

	"ndn-dpdk/app/inputdemux"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

// Input thread.
type Input struct {
	id     int
	demux3 inputdemux.Demux3
	rxl    *iface.RxLoop
}

func newInput(id int, lc dpdk.LCore, ndt *ndt.Ndt, fwds []*Fwd) *Input {
	socket := lc.GetNumaSocket()
	var fwi Input
	fwi.id = id

	fwi.demux3 = inputdemux.NewDemux3(socket)
	demuxI := fwi.demux3.GetInterestDemux()
	demuxI.InitNdt(ndt, id)
	demuxD := fwi.demux3.GetDataDemux()
	demuxD.InitToken()
	demuxN := fwi.demux3.GetNackDemux()
	demuxN.InitToken()
	for i, fwd := range fwds {
		demuxI.SetDest(i, fwd.interestQueue)
		demuxD.SetDest(i, fwd.dataQueue)
		demuxN.SetDest(i, fwd.nackQueue)
	}

	fwi.rxl = iface.NewRxLoop(socket)
	fwi.rxl.SetLCore(lc)
	fwi.rxl.SetCallback(inputdemux.Demux3_FaceRx, fwi.demux3.GetPtr())
	return &fwi
}

func (fwi *Input) Close() error {
	fwi.demux3.Close()
	return fwi.rxl.Close()
}

func (fwi *Input) String() string {
	return fmt.Sprintf("input%d", fwi.id)
}
