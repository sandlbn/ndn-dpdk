package strategycode

/*
#include "strategy-code.h"
*/
import "C"
import (
	"io/ioutil"
	"os"
	"unsafe"

	"ndn-dpdk/dpdk"
)

// External symbols available to eBPF programs, provided by ndn-dpdk/app/fwdp package.
var (
	Xsyms  unsafe.Pointer
	NXsyms int
)

func makeStrategyCode(name string, bpf *C.struct_rte_bpf) (sc *scImpl, e error) {
	if bpf == nil {
		return nil, dpdk.GetErrno()
	}

	var jit C.struct_rte_bpf_jit
	res := C.rte_bpf_get_jit_(bpf, &jit)
	if res != 0 {
		C.rte_bpf_destroy_(bpf)
		return nil, dpdk.Errno(-res)
	}

	tableLock.Lock()
	defer tableLock.Unlock()
	lastId++

	sc = new(scImpl)
	sc.c = (*C.StrategyCode)(dpdk.Zmalloc("StrategyCode", C.sizeof_StrategyCode, dpdk.NUMA_SOCKET_ANY))
	sc.c.id = C.int(lastId)
	sc.c.name = C.CString(name)
	sc.c.nRefs = 1
	sc.c.bpf = bpf
	sc.c.jit = jit._func
	table[lastId] = sc
	return sc, nil
}

var dotTextSection = C.CString(".text")

// Load a strategy BPF program from ELF object.
func Load(name string, elf []byte) (sc StrategyCode, e error) {
	file, e := ioutil.TempFile("", "strategy*.so")
	if e != nil {
		return nil, e
	}
	if _, e := file.Write(elf); e != nil {
		return nil, e
	}
	filename := file.Name()
	file.Close()
	defer os.Remove(filename)

	var prm C.struct_rte_bpf_prm
	prm.xsym = (*C.struct_rte_bpf_xsym)(Xsyms)
	prm.nb_xsym = (C.uint32_t)(NXsyms)
	prm.prog_arg._type = C.RTE_BPF_ARG_RAW

	filenameC := C.CString(filename)
	defer C.free(unsafe.Pointer(filenameC))
	bpf := C.rte_bpf_elf_load_(&prm, filenameC, dotTextSection)
	return makeStrategyCode(name, bpf)
}

// Load an empty BPF program (mainly for unit testing).
func MakeEmpty(name string) StrategyCode {
	var prm C.struct_rte_bpf_prm
	prm.ins = C.StrategyCode_GetEmptyProgram_(&prm.nb_ins)
	prm.prog_arg._type = C.RTE_BPF_ARG_RAW

	bpf := C.rte_bpf_load_(&prm)
	sc, e := makeStrategyCode(name, bpf)
	if e != nil {
		panic(e)
	}
	return sc
}
