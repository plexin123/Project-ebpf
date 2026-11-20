export type CallEvent = {
  caller: string;
  callee: string;
};

export type FunctionEvent = {
  funcName: string;
  duration: number;
  // only on live events; history rows from SQLite don't carry it
  selfTime?: number;
  baseline: number;
  current: number;
  driftPct: number;
  status: "ok" | "baseline_set" | "regression";
  timestamp: number;
  commithash: string;
};

export type WSMessage = {
  type: string;
  payload: FunctionEvent | CallEvent;
  traceId : string
  pid: number
  tid: number
};
