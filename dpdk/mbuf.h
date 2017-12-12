#ifndef NDN_TRAFFIC_DPDK_DPDK_MBUF_H
#define NDN_TRAFFIC_DPDK_DPDK_MBUF_H

#include "../common.h"
#include <rte_memcpy.h>
#include <rte_prefetch.h>

// Position iterator within a packet.
// It can optionally carry a boundary so that the iterator cannot be advanced
// past this limit.
typedef struct MbufLoc
{
  const struct rte_mbuf* m; // current segment
  uint32_t rem;             // remaining octets
  uint16_t off;             // offset within current segment
} MbufLoc;

// Initialize a MbufLoc to the beginning of a packet.
static inline void
MbufLoc_Init(MbufLoc* ml, const struct rte_mbuf* pkt)
{
  ml->m = pkt;
  ml->off = 0;
  ml->rem = pkt->pkt_len;
}

// Test if the iterator points past the end of packet or boundary.
static inline bool
MbufLoc_IsEnd(const MbufLoc* ml)
{
  return ml->m == NULL || ml->rem == 0;
}

// Advance the position by n octets.
static inline void
MbufLoc_Advance(MbufLoc* ml, uint32_t n)
{
  assert(!MbufLoc_IsEnd(ml));

  if (n > ml->rem) {
    ml->m = NULL;
    return;
  }

  uint32_t last = ml->off + n;
  while (unlikely(last >= ml->m->data_len)) {
    last -= ml->m->data_len;
    ml->m = ml->m->next;
    ml->off = 0;
    if (unlikely(ml->m == NULL)) {
      return;
    }
  }
  ml->off = (uint16_t)last;
}

// Determine the distance in octets from a to b.
// If MbufLoc_Diff(a, b) == n and n >= 0, it implies MbufLoc_Advance(a, n)
// equals b.
// If MbufLoc_Diff(a, b) == n and n <= 0, it implies MbufLoc_Advance(b, -n)
// equals a.
// Behavior is undefined if a and b do not point to the same packet.
// This function does not honor the iterator boundary.
ptrdiff_t MbufLoc_Diff(const MbufLoc* a, const MbufLoc* b);

extern uint32_t __MbufLoc_Read_MultiSeg(MbufLoc* ml, void* output, uint32_t n);

// Copy next n octets, and advance the position.
// Return number of octets copied.
static inline uint32_t
MbufLoc_Read(MbufLoc* ml, void* output, uint32_t n)
{
  assert(!MbufLoc_IsEnd(ml));

  if (n > ml->rem) {
    n = ml->rem;
  }
  ml->rem -= n;

  uint8_t* data = rte_pktmbuf_mtod_offset(ml->m, uint8_t*, ml->off);
  rte_prefetch0(data);

  uint32_t last = ml->off + n;
  if (unlikely(last >= ml->m->data_len)) {
    return __MbufLoc_Read_MultiSeg(ml, output, n);
  }

  rte_memcpy(output, data, n);
  ml->off = (uint16_t)last;
  return n;
}

static inline bool
MbufLoc_ReadU8(MbufLoc* ml, uint8_t* output)
{
  return sizeof(uint8_t) == MbufLoc_Read(ml, output, sizeof(uint8_t));
}

static inline bool
MbufLoc_ReadU16(MbufLoc* ml, uint16_t* output)
{
  return sizeof(uint16_t) == MbufLoc_Read(ml, output, sizeof(uint16_t));
}

static inline uint32_t
MbufLoc_ReadU32(MbufLoc* ml, uint32_t* output)
{
  return sizeof(uint32_t) == MbufLoc_Read(ml, output, sizeof(uint32_t));
}

static inline uint64_t
MbufLoc_ReadU64(MbufLoc* ml, uint64_t* output)
{
  return sizeof(uint64_t) == MbufLoc_Read(ml, output, sizeof(uint64_t));
}

#endif // NDN_TRAFFIC_DPDK_DPDK_