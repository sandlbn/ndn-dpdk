import * as pktqueue from "../../container/pktqueue/mod";
import { Counter } from "../../core/mod";
import { Nanoseconds } from "../../core/nnduration/mod";

export interface Config {
  /**
   * @TJS-type integer
   * @minimum 1
   * @default 1
   */
  NThreads?: number;

  /**
   * @TJS-type integer
   * @minimum 1
   * @default 1
   */
  NProcs?: number;

  RxQueue?: pktqueue.Config;

  /**
   * @TJS-type integer
   * @minimum 1
   * @default 65536
   */
  WindowCapacity?: number;
}

export interface Counters {
  Time: unknown;
  LastRtt: Nanoseconds;
  SRtt: Nanoseconds;
  Rto: Nanoseconds;
  Cwnd: Counter;
  NInFlight: Counter;
  NTxRetx: Counter;
  NRxData: Counter;
}
