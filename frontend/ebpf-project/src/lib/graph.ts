
    function add_edge(state: TraceState , callEvent: CallEvent) : TraceState{
        const newState = new Map(state)
        let currentGraphState =  newState.get(callEvent.traceId)
        if (!currentGraphState){
            currentGraphState = {
                edges: new Map<string, string[]>(),
                node: new Map<string, FunctionEvent[]>()
            }
            newState.set(callEvent.traceId, currentGraphState)
        }
        else {
              currentGraphState = {
                edges : new Map(currentGraphState.edges),
                node : new Map(currentGraphState.node)
            }
            newState.set(callEvent.traceId, currentGraphState)
        }
        let father = callEvent.caller
        let children =callEvent.calle
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


    function upsert_node(state: TraceState, functionEvent: FunctionEvent, traceId: number): TraceState{
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


