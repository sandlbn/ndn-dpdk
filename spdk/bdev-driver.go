package spdk

/*
#include <spdk/copy_engine.h>
*/
import "C"
import (
	"fmt"
	"sync"

	"ndn-dpdk/dpdk"
)

var initCopyEngineOnce sync.Once

func initCopyEngine() {
	initCopyEngineOnce.Do(func() {
		MainThread.Call(func() { C.spdk_copy_engine_initialize() })
	})
}

type constructMallocBdevArgs struct {
	BlockSize int `json:"block_size"`
	NumBlocks int `json:"num_blocks"`
}

// Create Malloc block device.
func NewMallocBdev(blockSize int, nBlocks int) (bdi BdevInfo, e error) {
	initCopyEngine() // Malloc bdev depends on copy engine
	var args constructMallocBdevArgs
	args.BlockSize = blockSize
	args.NumBlocks = nBlocks
	var name string
	if e = RpcCall("construct_malloc_bdev", args, &name); e != nil {
		return BdevInfo{}, e
	}
	return mustFindBdev(name), nil
}

type deleteMallocBdevArgs struct {
	Name string `json:"name"`
}

// Destroy Malloc block device.
func DestroyMallocBdev(bdi BdevInfo) (e error) {
	var args deleteMallocBdevArgs
	args.Name = bdi.GetName()
	var ok bool
	return RpcCall("delete_malloc_bdev", args, &ok)
}

var lastAioBdevId int

type constructAioBdevArgs struct {
	Name      string `json:"name"`
	Filename  string `json:"filename"`
	BlockSize int    `json:"block_size,omitempty"`
}

// Create Linux AIO block device.
func NewAioBdev(filename string, blockSize int) (bdi BdevInfo, e error) {
	var args constructAioBdevArgs
	lastAioBdevId++
	args.Name = fmt.Sprintf("Aio%d", lastAioBdevId)
	args.Filename = filename
	args.BlockSize = blockSize
	var name string
	if e = RpcCall("construct_aio_bdev", args, &name); e != nil {
		return BdevInfo{}, e
	}
	return mustFindBdev(name), nil
}

type deleteAioBdevArgs struct {
	Name string `json:"name"`
}

// Destroy Linux AIO block device.
func DestroyAioBdev(bdi BdevInfo) (e error) {
	var args deleteAioBdevArgs
	args.Name = bdi.GetName()
	var ok bool
	return RpcCall("delete_aio_bdev", args, &ok)
}

func makeNvmeName(pciAddr dpdk.PciAddress) string {
	return fmt.Sprintf("nvme%02x%02x%01x", pciAddr.Bus, pciAddr.Devid, pciAddr.Function)
}

type constructNvmeBdevArgs struct {
	Name   string `json:"name"`
	TrType string `json:"trtype"`
	TrAddr string `json:"traddr"`
}

func AttachNvmeBdevs(pciAddr dpdk.PciAddress) (bdis []BdevInfo, e error) {
	var args constructNvmeBdevArgs
	args.Name = makeNvmeName(pciAddr)
	args.TrType = "pcie"
	args.TrAddr = pciAddr.String()

	var namespaces []string
	if e = RpcCall("construct_nvme_bdev", args, &namespaces); e != nil {
		return nil, e
	}

	for _, namespace := range namespaces {
		bdis = append(bdis, mustFindBdev(namespace))
	}
	return bdis, nil
}

type deleteNvmeControllerArgs struct {
	Name string `json:"name"`
}

func DetachNvmeBdevs(pciAddr dpdk.PciAddress) (e error) {
	var args deleteNvmeControllerArgs
	args.Name = makeNvmeName(pciAddr)
	var ok bool
	return RpcCall("delete_nvme_controller", args, &ok)
}
