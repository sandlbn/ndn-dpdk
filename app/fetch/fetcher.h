#ifndef NDN_DPDK_APP_FETCH_FETCHER_H
#define NDN_DPDK_APP_FETCH_FETCHER_H

/// \file

#include "../../container/pktqueue/queue.h"
#include "../../dpdk/thread.h"
#include "../../iface/face.h"
#include "logic.h"

/** \brief Fetch procedure that fetches from one prefix.
 */
typedef struct FetchProc
{
  struct cds_hlist_node fthNode;
  PktQueue rxQueue;
  FetchLogic logic;
  uint64_t pitToken;
  InterestTemplate tpl;
} FetchProc;

/** \brief Fetch thread that runs several fetch procedures.
 */
typedef struct FetchThread
{
  struct rte_mempool* interestMp;
  struct cds_hlist_head head;
  NonceGen nonceGen;
  FaceId face;
  ThreadStopFlag stop;
} FetchThread;

int
FetchThread_Run(FetchThread* fth);

#endif // NDN_DPDK_APP_FETCH_FETCHER_H
