import type { CallEvent, FunctionEvent } from '../types/events'
import { add_edge, upsert_node } from '../lib/graph'

export type GraphState = {
  edges: Map<string, string[]>;
  node: Map<string, FunctionEvent[]>;
  // process that produced this trace (a trace never spans two processes)
  pid?: number;
};

export type TracesState = Map<string, GraphState>

export type GraphAction =
  | { type: 'ADD_EDGE'; payload: CallEvent; traceId: string; pid: number }
  | { type: 'UPSERT_NODE'; payload: FunctionEvent; traceId: string; pid: number }

export const initialTracesState: TracesState = new Map()

export function tracesReducer(state: TracesState, action: GraphAction): TracesState {
    switch (action.type) {
        case 'ADD_EDGE': return add_edge(state, action.payload , action.traceId, action.pid)
        case 'UPSERT_NODE': return upsert_node(state, action.payload, action.traceId, action.pid)
    }
}
