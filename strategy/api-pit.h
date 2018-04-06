#ifndef NDN_DPDK_STRATEGY_API_PIT_H
#define NDN_DPDK_STRATEGY_API_PIT_H

/// \file

#include "../core/common1.h"

typedef uint16_t FaceId;
typedef uint64_t TscTime;
typedef int64_t TscDuration;

typedef struct SgPitDn
{
  TscTime expiry;
  char _a[12];
  FaceId face;
} __rte_aligned(32) SgPitDn;

typedef struct SgPitUp
{
  char _a[4];
  FaceId face;
  char _b[1];
  uint8_t nack;

  TscTime lastTx;
  TscDuration suppress;
  uint16_t nTx;
} __rte_aligned(64) SgPitUp;

#define SG_PIT_ENTRY_MAX_DNS 6
#define SG_PIT_ENTRY_MAX_UPS 2
#define SG_PIT_ENTRY_EXT_MAX_DNS 72
#define SG_PIT_ENTRY_EXT_MAX_UPS 36

typedef struct SgPitEntryExt SgPitEntryExt;

typedef struct SgPitEntry
{
  char _a[40];
  SgPitEntryExt* ext;
  SgPitDn dns[SG_PIT_ENTRY_MAX_DNS];
  SgPitUp ups[SG_PIT_ENTRY_MAX_UPS];
} SgPitEntry;

struct SgPitEntryExt
{
  SgPitDn dns[SG_PIT_ENTRY_EXT_MAX_DNS];
  SgPitUp ups[SG_PIT_ENTRY_EXT_MAX_UPS];
  SgPitEntryExt* next;
};

#endif // NDN_DPDK_STRATEGY_API_PIT_H
