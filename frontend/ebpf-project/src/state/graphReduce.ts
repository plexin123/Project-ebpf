import type { CallEvent, FunctionEvent } from '../types/events'
import { add_edge, upsert_node } from '../lib/graph'

export type GraphState = {
  edges: Map<string, string[]>;
  node: Map<string, FunctionEvent[]>;
};

export type TracesState = Map<string, GraphState>

export type GraphAction =
  | { type: 'ADD_EDGE'; payload: CallEvent; traceId: string }
  | { type: 'UPSERT_NODE'; payload: FunctionEvent; traceId: string }

export const initialTracesState: TracesState = new Map()

export function tracesReducer(state: TracesState, action: GraphAction): TracesState {
    switch (action.type) {
        case 'ADD_EDGE': return add_edge(state, action.payload , action.traceId)
        case 'UPSERT_NODE': return upsert_node(state, action.payload, action.traceId)
    }
}
