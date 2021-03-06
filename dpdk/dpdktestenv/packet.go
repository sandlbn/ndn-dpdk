package dpdktestenv

import (
	"encoding/hex"
	"fmt"
	"strings"

	"ndn-dpdk/dpdk"
)

// Make packet from byte slice(s), each slice becomes a segment.
// Memory is allocated from DirectMp.
// Caller is responsible for closing the packet.
func PacketFromBytes(inputs ...[]byte) (pkt dpdk.Packet) {
	return packetFromBytesInMp(MPID_DIRECT, inputs...)
}

func packetFromBytesInMp(mpid string, inputs ...[]byte) (pkt dpdk.Packet) {
	if len(inputs) == 0 {
		return Alloc(mpid).AsPacket()
	}

	mbufs := make([]dpdk.Mbuf, len(inputs))
	AllocBulk(mpid, mbufs)
	pkt = mbufs[0].AsPacket()
	seg := pkt.GetFirstSegment()
	for i, m := range mbufs {
		var e error
		if i > 0 {
			seg, e = pkt.AppendSegmentHint(m, &seg)
			if e != nil {
				panic(fmt.Sprintf("Packet.AppendSegment error %v, packet too long?", e))
			}
		}
		seg.SetHeadroom(0)
		e = seg.Append(inputs[i])
		if e != nil {
			panic(fmt.Sprintf("Segment.Append error %v, packet too long?", e))
		}
	}

	return pkt
}

func BytesFromHex(input string) []byte {
	s := strings.Map(func(ch rune) rune {
		if strings.ContainsRune("0123456789ABCDEF", ch) {
			return ch
		}
		return -1
	}, input)
	decoded, e := hex.DecodeString(s)
	if e != nil {
		panic(fmt.Sprintf("hex.DecodeString error %v", e))
	}
	return decoded
}

// Make packet from hexadecimal string(s), each string becomes a segment.
// The octets must be written as upper case.
// All characters other than [0-9A-F] are considered as comments and stripped.
func PacketFromHex(inputs ...string) dpdk.Packet {
	byteSlices := make([][]byte, len(inputs))
	for i, input := range inputs {
		byteSlices[i] = BytesFromHex(input)
	}
	return PacketFromBytes(byteSlices...)
}

// Split last n octets of last segment into a separate segment.
func PacketSplitTailSegment(pkt dpdk.Packet, n int) dpdk.Packet {
	seg := pkt.GetLastSegment()
	segLen := seg.Len()
	if segLen <= n {
		panic(fmt.Errorf("last segment has %d octets, cannot remove %d octets", segLen, n))
	}
	segBytes := seg.AsByteSlice()
	tail := PacketFromBytes(segBytes[segLen-n:])
	for i := segLen - n; i < segLen; i++ {
		segBytes[i]++
	}
	seg.Trim(n)
	pkt.AppendPacket(tail)
	return pkt
}
