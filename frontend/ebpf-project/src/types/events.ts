export type CallEvent = {
  caller: string;
  callee: string;
};

export type FunctionEvent = {
  funcName: string;
  duration: number;
  baseline: number;
  current: number;
  driftPct: number;
  status: "ok" | "baseline_set" | "regression";
};

export type WSMessage = {
  type: string;
  payload: FunctionEvent | CallEvent;
  traceId : string
};
