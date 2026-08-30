import type { CallEvent, FunctionEvent } from '../types/events'
import type { GraphState, TracesState } from '../state/graphReduce'

    export function add_edge(state: TracesState , callEvent: CallEvent, traceId: string) : TracesState{
        const newState = new Map(state)
        let currentGraphState =  newState.get(traceId)
        if (!currentGraphState){
            currentGraphState = {
                edges: new Map<string, string[]>(),
                node: new Map<string, FunctionEvent[]>()
            }
            newState.set(traceId, currentGraphState)
        }
        else {
              currentGraphState = {
                edges : new Map(currentGraphState.edges),
                node : new Map(currentGraphState.node)
            }
            newState.set(traceId, currentGraphState)
        }
        let father = callEvent.caller
        let children = callEvent.callee
        if (!currentGraphState.edges.get(father)){
            currentGraphState.edges.set(father, [])
        }
        else{
            const listChildren = Array.from(currentGraphState.edges.get(father)!)
            currentGraphState.edges.set(father, listChildren)
        }
        currentGraphState.edges.get(father)?.push(children)
        return newState
    }


    export function upsert_node(state: TracesState, functionEvent: FunctionEvent, traceId: string): TracesState{
        const newState = new Map(state)
        let currentGraphState =  newState.get(traceId)
        if (!currentGraphState){
            currentGraphState = {
                edges: new Map<string, string[]>(),
                node: new Map<string, FunctionEvent[]>()
            }   
            newState.set(traceId, currentGraphState)
        }
        else {
            currentGraphState = {
                edges : new Map(currentGraphState.edges),
                node : new Map(currentGraphState.node)
            }
            newState.set(traceId, currentGraphState)
        }
        let funcName =  functionEvent.funcName
        if (!currentGraphState.node.get(funcName)){
            currentGraphState.node.set(funcName, [])
        }
        else{
            const listEvent = Array.from(currentGraphState.node.get(funcName)!)
            currentGraphState.node.set(funcName, listEvent)
        }
        currentGraphState.node.get(funcName)?.push(functionEvent)
        return newState

    }

export type GraphNode = {
    id: string
    status: FunctionEvent['status'] | 'unknown'
    duration: number
    driftPct: number
    callCount: number
}

export type GraphLink = {
    id: string
    source: string
    target: string
}

export function mergeTraces(traces: TracesState): GraphState {
    const merged: GraphState = {
        edges: new Map<string, string[]>(),
        node: new Map<string, FunctionEvent[]>()
    }
    for (const graphState of traces.values()) {
        for (const [father, children] of graphState.edges) {
            const existing = merged.edges.get(father) ?? []
            merged.edges.set(father, [...existing, ...children])
        }
        for (const [funcName, history] of graphState.node) {
            const existing = merged.node.get(funcName) ?? []
            merged.node.set(funcName, [...existing, ...history])
        }
    }
    return merged
}

export function toGraphNodes(graphState: GraphState | undefined): GraphNode[] {
    if (!graphState) return []
    const nodes = new Map<string, GraphNode>()

    for (const [funcName, history] of graphState.node) {
        const latest = history[history.length - 1]
        nodes.set(funcName, {
            id: funcName,
            status: latest.status,
            duration: latest.duration,
            driftPct: latest.driftPct,
            callCount: history.length
        })
    }

    // a function can appear as caller/callee before its own FunctionEvent arrives
    for (const [father, children] of graphState.edges) {
        if (!nodes.has(father)) {
            nodes.set(father, { id: father, status: 'unknown', duration: 0, driftPct: 0, callCount: 0 })
        }
        for (const child of children) {
            if (!nodes.has(child)) {
                nodes.set(child, { id: child, status: 'unknown', duration: 0, driftPct: 0, callCount: 0 })
            }
        }
    }

    return Array.from(nodes.values())
}

export function toGraphLinks(graphState: GraphState | undefined): GraphLink[] {
    if (!graphState) return []
    const links: GraphLink[] = []
    for (const [father, children] of graphState.edges) {
        for (const child of children) {
            links.push({ id: `${father}->${child}`, source: father, target: child })
        }
    }
    return links
}

