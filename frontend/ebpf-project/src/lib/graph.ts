import type { CallEvent, FunctionEvent } from '../types/events'
import type { GraphState, TracesState } from '../state/graphReduce'

    export function add_edge(state: TracesState , callEvent: CallEvent, traceId: string, pid: number) : TracesState{
        const newState = new Map(state)
        let currentGraphState =  newState.get(traceId)
        if (!currentGraphState){
            currentGraphState = {
                edges: new Map<string, string[]>(),
                node: new Map<string, FunctionEvent[]>(),
                pid
            }
            newState.set(traceId, currentGraphState)
        }
        else {
              currentGraphState = {
                edges : new Map(currentGraphState.edges),
                node : new Map(currentGraphState.node),
                pid : currentGraphState.pid
            }
            newState.set(traceId, currentGraphState)
        }
        let father = callEvent.caller
        let child = callEvent.callee
        const existingChildren = currentGraphState.edges.get(father) ?? []
        // edges son relaciones únicas (¿existe esta llamada, sí o no?), no un
        // historial de ocurrencias -- si ya está, no la duplicamos
        if (!existingChildren.includes(child)) {
            currentGraphState.edges.set(father, [...existingChildren, child])
        }
        return newState
    }


    export function upsert_node(state: TracesState, functionEvent: FunctionEvent, traceId: string, pid: number): TracesState{
        const newState = new Map(state)
        let currentGraphState =  newState.get(traceId)
        if (!currentGraphState){
            currentGraphState = {
                edges: new Map<string, string[]>(),
                node: new Map<string, FunctionEvent[]>(),
                pid
            }
            newState.set(traceId, currentGraphState)
        }
        else {
            currentGraphState = {
                edges : new Map(currentGraphState.edges),
                node : new Map(currentGraphState.node),
                pid : currentGraphState.pid
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
    // `${pid}:${name}` -- the same function in two processes is two nodes
    id: string
    name: string
    pid: number
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
    return mergeGraphs(traces.values())
}

// one merged graph per process, so the call graph can draw a circle around each
export function mergeTracesByPid(traces: TracesState): Map<number, GraphState> {
    const groups = new Map<number, GraphState[]>()
    for (const graphState of traces.values()) {
        const pid = graphState.pid ?? 0
        const group = groups.get(pid) ?? []
        group.push(graphState)
        groups.set(pid, group)
    }
    const byPid = new Map<number, GraphState>()
    for (const [pid, group] of groups) {
        byPid.set(pid, { ...mergeGraphs(group), pid })
    }
    return byPid
}

function mergeGraphs(graphs: Iterable<GraphState>): GraphState {
    const merged: GraphState = {
        edges: new Map<string, string[]>(),
        node: new Map<string, FunctionEvent[]>()
    }
    for (const graphState of graphs) {
        for (const [father, children] of graphState.edges) {
            const existing = merged.edges.get(father) ?? []
            for(const child of children){
                if (!existing.includes(child )){
                    existing.push(child)
                }
            }   
            merged.edges.set(father, [...existing])
        }
        for (const [funcName, history] of graphState.node) {
            const existing = merged.node.get(funcName) ?? []
            merged.node.set(funcName, [...existing, ...history])
        }
    }
    return merged
}

export function nodeId(pid: number, funcName: string): string {
    return `${pid}:${funcName}`
}

export function toGraphNodes(graphState: GraphState | undefined, pid: number): GraphNode[] {
    if (!graphState) return []
    const nodes = new Map<string, GraphNode>()
    const unknown = (name: string): GraphNode =>
        ({ id: nodeId(pid, name), name, pid, status: 'unknown', duration: 0, driftPct: 0, callCount: 0 })

    for (const [funcName, history] of graphState.node) {
        const latest = history[history.length - 1]
        nodes.set(funcName, {
            id: nodeId(pid, funcName),
            name: funcName,
            pid,
            status: latest.status,
            duration: latest.duration,
            driftPct: latest.driftPct,
            callCount: history.length
        })
    }

    // a function can appear as caller/callee before its own FunctionEvent arrives
    for (const [father, children] of graphState.edges) {
        if (!nodes.has(father)) {
            nodes.set(father, unknown(father))
        }
        for (const child of children) {
            if (!nodes.has(child)) {
                nodes.set(child, unknown(child))
            }
        }
    }

    return Array.from(nodes.values())
}

export function toGraphLinks(graphState: GraphState | undefined, pid: number): GraphLink[] {
    if (!graphState) return []
    const links: GraphLink[] = []
    for (const [father, children] of graphState.edges) {
        for (const child of children) {
            const source = nodeId(pid, father)
            const target = nodeId(pid, child)
            links.push({ id: `${source}->${target}`, source, target })
        }
    }
    return links
}

