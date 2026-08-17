type CallEvent = {
  traceId: number;
  caller: string;
  calle: string;
};

type GraphState = {
  edges: Map<string, string[]>;
  node: Map<string, FunctionEvent[]>;
};

type FunctionEvent = {
  funcName: string;
  duration: number;
  baseline: number  ;
  current: number;
  driftpct: number | undefined;
  status: "BaselineSet" | "Regression" | "Ok";
};

type WSMessage = {
  type: string;
  payload: FunctionEvent | CallEvent;
};


