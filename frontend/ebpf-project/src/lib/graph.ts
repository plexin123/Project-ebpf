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

// one function in a thread's call tree; the same function can appear under
// several parents (e.g. checksum under validate_order and save_order)
export type CallTreeNode = {
    name: string
    latest?: FunctionEvent
    // calls itself or an ancestor: drawn as a badge instead of an endless branch
    recursive: boolean
    children: CallTreeNode[]
}

export type ThreadLane = {
    tid: number
    roots: CallTreeNode[]
}

export type ProcessView = {
    pid: number
    label: string
    threads: ThreadLane[]
}

export function mergeTraces(traces: TracesState): GraphState {
    return mergeGraphs(traces.values())
}

// process -> thread -> call tree, sorted by pid/tid so the layout doesn't jump around
export function buildProcessViews(traces: TracesState): ProcessView[] {
    const byProcess = new Map<number, Map<number, GraphState[]>>()
    for (const graphState of traces.values()) {
        const pid = graphState.pid ?? 0
        const tid = graphState.tid ?? 0
        const threads = byProcess.get(pid) ?? new Map<number, GraphState[]>()
        const group = threads.get(tid) ?? []
        group.push(graphState)
        threads.set(tid, group)
        byProcess.set(pid, threads)
    }

    const views: ProcessView[] = []
    for (const [pid, threads] of byProcess) {
        const lanes: ThreadLane[] = []
        for (const [tid, graphs] of threads) {
            lanes.push({ tid, roots: callTrees(mergeGraphs(graphs)) })
        }
        lanes.sort((a, b) => a.tid - b.tid)
        views.push({ pid, label: processLabel(pid, lanes), threads: lanes })
    }
    return views.sort((a, b) => a.pid - b.pid)
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

// roots are the functions nobody calls inside this thread (the backend drops
// the "" -> root edge, so a root only shows up as a node or as a caller)
function callTrees(graph: GraphState): CallTreeNode[] {
    const names = new Set<string>(graph.node.keys())
    const callees = new Set<string>()
    for (const [caller, children] of graph.edges) {
        names.add(caller)
        for (const child of children) {
            names.add(child)
            if (child !== caller) callees.add(child)
        }
    }
    let roots = [...names].filter((name) => !callees.has(name))
    // everything is part of a cycle: start anywhere
    if (roots.length === 0) roots = [...names].slice(0, 1)

    const build = (name: string, path: Set<string>): CallTreeNode => {
        const history = graph.node.get(name)
        const node: CallTreeNode = { name, latest: history?.[history.length - 1], recursive: false, children: [] }
        path.add(name)
        for (const child of graph.edges.get(name) ?? []) {
            if (path.has(child)) {
                node.recursive = true
                continue
            }
            node.children.push(build(child, path))
        }
        path.delete(name)
        return node
    }
    return roots.map((root) => build(root, new Set()))
}

// "orders · pid 4121": the crate name comes from the Rust symbol prefix (orders::...)
function processLabel(pid: number, lanes: ThreadLane[]): string {
    for (const lane of lanes) {
        for (const root of lane.roots) {
            const sep = root.name.indexOf('::')
            if (sep > 0) return `${root.name.slice(0, sep)} · pid ${pid}`
        }
    }
    return `pid ${pid}`
}
