
function add_edge(callEvent: CallEvent){
    let currentGraphState =  dictionaryGraphState.get(callEvent.traceId)
    if (!currentGraphState){
        currentGraphState = {
            edges: new Map<string, string[]>(),
            node: new Map<string, FunctionEvent[]>()
        }
        dictionaryGraphState.set(callEvent.traceId, currentGraphState)
    }
    let father = callEvent.caller
    let children =callEvent.calle
    currentGraphState.edges.get(father)?.push(children)
    const newState = new Map(dictionaryGraphState)
    console.log(newState.get(callEvent.traceId)?.edges.get(father))
    return newState
}


function upsert_node(functionEvent: FunctionEvent, traceId: number){
    let currentGraphState =  dictionaryGraphState.get(traceId)
    if (!currentGraphState){
        currentGraphState = {
            edges: new Map<string, string[]>(),
            node: new Map<string, FunctionEvent[]>()
        }
         dictionaryGraphState.set(traceId, currentGraphState)
    let funcName =  functionEvent.funcName
    currentGraphState.node.get(funcName)?.push(functionEvent)
    //add the FunctionEvent for the string append structure,to keep track of the history of that function for that specific traceId
}
}


