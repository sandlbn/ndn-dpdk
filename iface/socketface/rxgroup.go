package socketface

/*
#include "socket-face.h"

void
c_SocketFace_PostRxBurst(SocketFace* face, void* burst, uint16_t nRx, void* cb, void* cbarg)
{
	FaceImpl_RxBurst(&face->base, (FaceRxBurst*)burst, nRx, (Face_RxCb)cb, cbarg);
}
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/iface"
)

type RxGroup struct {
	faces     map[*SocketFace]struct{}
	running   bool
	closeCmd  chan struct{}
	addCmd    chan *SocketFace
	removeCmd chan *SocketFace
}

func NewRxGroup(faces ...*SocketFace) (rxg RxGroup) {
	rxg.faces = make(map[*SocketFace]struct{})
	for _, face := range faces {
		rxg.faces[face] = struct{}{}
	}

	rxg.closeCmd = make(chan struct{})
	rxg.addCmd = make(chan *SocketFace)
	rxg.removeCmd = make(chan *SocketFace)
	return rxg
}

func (rxg RxGroup) Close() error {
	rxg.closeCmd <- struct{}{}
	return nil
}

func (rxg RxGroup) AddFace(face *SocketFace) {
	rxg.addCmd <- face
}

func (rxg RxGroup) RemoveFace(face *SocketFace) {
	rxg.removeCmd <- face
}

func (rxg RxGroup) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	burst := iface.NewRxBurst(burstSize)
	for {
		select {
		case <-rxg.closeCmd:
			return
		case face := <-rxg.addCmd:
			rxg.faces[face] = struct{}{}
		case face := <-rxg.removeCmd:
			delete(rxg.faces, face)
		default:
		}

		for face := range rxg.faces {
			nRx := face.rxBurst(burst)
			if nRx > 0 {
				C.c_SocketFace_PostRxBurst(face.getPtr(), burst.GetPtr(), C.uint16_t(nRx), cb, cbarg)
			}
		}
	}
}