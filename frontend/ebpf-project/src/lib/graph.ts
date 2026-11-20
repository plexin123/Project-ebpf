import type { CallEvent, FunctionEvent } from '../types/events'
import type { GraphState, TracesState } from '../state/graphReduce'

    export function add_edge(state: TracesState , callEvent: CallEvent, traceId: string, pid: number, tid: number) : TracesState{
        const newState = new Map(state)
        let currentGraphState =  newState.get(traceId)
        if (!currentGraphState){
            currentGraphState = {
                edges: new Map<string, string[]>(),
                node: new Map<string, FunctionEvent[]>(),
                pid,
                tid
            }
            newState.set(traceId, currentGraphState)
        }
        else {
              currentGraphState = {
                edges : new Map(currentGraphState.edges),
                node : new Map(currentGraphState.node),
                pid : currentGraphState.pid,
                tid : currentGraphState.tid
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


    export function upsert_node(state: TracesState, functionEvent: FunctionEvent, traceId: string, pid: number, tid: number): TracesState{
        const newState = new Map(state)
        let currentGraphState =  newState.get(traceId)
        if (!currentGraphState){
            currentGraphState = {
                edges: new Map<string, string[]>(),
                node: new Map<string, FunctionEvent[]>(),
                pid,
                tid
            }
            newState.set(traceId, currentGraphState)
        }
        else {
            currentGraphState = {
                edges : new Map(currentGraphState.edges),
                node : new Map(currentGraphState.node),
                pid : currentGraphState.pid,
                tid : currentGraphState.tid
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

// one card in the call graph: every function appears once per process
export type FlowNode = {
    name: string
    latest?: FunctionEvent
    // calls itself: shown as a badge, the self-edge is not drawn
    recursive: boolean
}

export type FlowEdge = {
    source: string
    target: string
}

// all threads of a process merged into one graph
export type ProcessGraph = {
    pid: number
    label: string
    threadCount: number
    nodes: FlowNode[]
    edges: FlowEdge[]
}

export function mergeTraces(traces: TracesState): GraphState {
    return mergeGraphs(traces.values())
}

// sorted by pid so the boxes don't jump around as events arrive
export function buildProcessGraphs(traces: TracesState): ProcessGraph[] {
    const byProcess = new Map<number, { graphs: GraphState[]; tids: Set<number> }>()
    for (const graphState of traces.values()) {
        const pid = graphState.pid ?? 0
        const group = byProcess.get(pid) ?? { graphs: [], tids: new Set<number>() }
        group.graphs.push(graphState)
        group.tids.add(graphState.tid ?? 0)
        byProcess.set(pid, group)
    }

    const processes: ProcessGraph[] = []
    for (const [pid, { graphs, tids }] of byProcess) {
        const merged = mergeGraphs(graphs)
        const nodes = new Map<string, FlowNode>()
        const nodeFor = (name: string): FlowNode => {
            let node = nodes.get(name)
            if (!node) {
                node = { name, latest: newest(merged.node.get(name)), recursive: false }
                nodes.set(name, node)
            }
            return node
        }

        for (const name of merged.node.keys()) nodeFor(name)
        const edges: FlowEdge[] = []
        for (const [caller, callees] of merged.edges) {
            const source = nodeFor(caller)
            for (const callee of callees) {
                if (callee === caller) {
                    source.recursive = true
                    continue
                }
                nodeFor(callee)
                edges.push({ source: caller, target: callee })
            }
        }

        processes.push({
            pid,
            label: processLabel(pid, [...nodes.keys()]),
            threadCount: tids.size,
            nodes: [...nodes.values()],
            edges
        })
    }
    return processes.sort((a, b) => a.pid - b.pid)
}

// history is concatenated from several threads, so the last element isn't always the newest
function newest(history: FunctionEvent[] | undefined): FunctionEvent | undefined {
    if (!history || history.length === 0) return undefined
    return history.reduce((a, b) => (b.timestamp >= a.timestamp ? b : a))
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

// "orders · pid 4121": the crate name comes from the Rust symbol prefix (orders::...)
function processLabel(pid: number, names: string[]): string {
    for (const name of names) {
        const sep = name.indexOf('::')
        if (sep > 0) return `${name.slice(0, sep)} · pid ${pid}`
    }
    return `pid ${pid}`
}
