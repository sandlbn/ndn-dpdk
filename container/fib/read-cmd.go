package fib

/*
#include "fib.h"
*/
import "C"
import (
	"ndn-dpdk/container/fib/fibtree"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/ndn"
)

// List all FIB entry names.
func (fib *Fib) ListNames() (names []*ndn.Name) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		fib.tree.Traverse(func(name *ndn.Name, n *fibtree.Node) bool {
			if n.IsEntry {
				names = append(names, name)
			}
			return true
		})
		return nil
	})
	return names
}

// Perform an exact match lookup.
func (fib *Fib) Find(name *ndn.Name) (entry *Entry) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		_, partition := fib.ndt.Lookup(name)
		entry = fib.FindInPartition(name, int(partition), rs)
		return nil
	})
	return entry
}

// Perform an exact match lookup in specified partition.
// This method runs in the given URCU read-side thread, not necessarily the command loop.
func (fib *Fib) FindInPartition(name *ndn.Name, partition int, rs *urcu.ReadSide) (entry *Entry) {
	rs.Lock()
	defer rs.Unlock()
	nameV := name.GetValue()
	hash := name.ComputeHash()
	return entryFromC(C.Fib_Find_(fib.parts[partition].c, C.uint16_t(len(nameV)),
		(*C.uint8_t)(nameV.GetPtr()), C.uint64_t(hash)))
}

// Determine what partitions would a name appear in.
// This method is non-thread-safe.
func (fib *Fib) listPartitionsForName(name *ndn.Name) (parts []*partition) {
	if name.Len() < fib.ndt.GetPrefixLen() {
		return fib.parts
	}
	_, partition := fib.ndt.Lookup(name)
	return append(parts, fib.parts[partition])
}

// Read entry counters, aggregate across all partitions if necessary.
func (fib *Fib) ReadEntryCounters(name *ndn.Name) (cnt EntryCounters) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		for _, part := range fib.listPartitionsForName(name) {
			if entry := fib.FindInPartition(name, part.index, rs); entry != nil {
				cnt.Add(entry)
			}
		}
		return nil
	})
	return cnt
}

// Perform a longest prefix match lookup.
func (fib *Fib) Lpm(name *ndn.Name) (entry *Entry) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		_, partition := fib.ndt.Lookup(name)
		entry = entryFromC(C.Fib_Lpm_(fib.parts[partition].c, (*C.PName)(name.GetPNamePtr()),
			(*C.uint8_t)(name.GetValue().GetPtr())))
		return nil
	})
	return entry
}
