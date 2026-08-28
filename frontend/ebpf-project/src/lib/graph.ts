import type { CallEvent, FunctionEvent } from '../types/events'
import type { TracesState } from '../state/graphReduce'

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


