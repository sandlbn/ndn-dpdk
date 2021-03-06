package pit

/*
#include "../pcct/pit-up.h"
*/
import "C"
import (
	"ndn-dpdk/iface"
)

// A PIT upstream record.
type Up struct {
	c     *C.PitUp
	entry Entry
}

func (up Up) GetFaceId() iface.FaceId {
	return iface.FaceId(up.c.face)
}
