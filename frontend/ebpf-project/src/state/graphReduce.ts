import type { CallEvent, FunctionEvent } from '../types/events'
import { add_edge, upsert_node } from '../lib/graph'

export type GraphState = {
  edges: Map<string, string[]>;
  node: Map<string, FunctionEvent[]>;
  // process and thread that produced this trace (a trace never spans two threads)
  pid?: number;
  tid?: number;
};

export type TracesState = Map<string, GraphState>

export type GraphAction =
  | { type: 'ADD_EDGE'; payload: CallEvent; traceId: string; pid: number; tid: number }
  | { type: 'UPSERT_NODE'; payload: FunctionEvent; traceId: string; pid: number; tid: number }

export const initialTracesState: TracesState = new Map()

export function tracesReducer(state: TracesState, action: GraphAction): TracesState {
    switch (action.type) {
        case 'ADD_EDGE': return add_edge(state, action.payload , action.traceId, action.pid, action.tid)
        case 'UPSERT_NODE': return upsert_node(state, action.payload, action.traceId, action.pid, action.tid)
    }
}
