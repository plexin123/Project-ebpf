    // showcase the graph first in a table and then a graph node


    // this is going to connect to the backend using websockets
    // then create a graph using the data received from the backend
    // then display the graph in a table and then a graph node

    import React, { useEffect, useReducer, useState } from 'react'

    import { WebsocketConnectionEntrance } from '../hooks/useWebsockets'
    import { useGraph } from '../state/GraphContext'
    import { CallEvent, FunctionEvent, WSMessage } from '../types/events'


    const DEFAULT_URL = ""

    export function Dashboard() {
        const {traces,  dispatch} = useGraph()
        const [traceId, setTraceId] = useState("")
        const status = WebsocketConnectionEntrance(DEFAULT_URL,(ws_message : WSMessage) =>{     
            setTraceId(ws_message.traceId)
            switch(ws_message?.type){
                case("connection"):
                    let callEvent: CallEvent = ws_message.payload as CallEvent
                    dispatch({type: 'ADD_EDGE', payload: callEvent, traceId: ws_message.traceId })
                    break
                case("event"):
                    let functionEvent : FunctionEvent = ws_message.payload as FunctionEvent
                    dispatch({type: 'UPSERT_NODE' , payload: functionEvent , traceId: ws_message.traceId })
                    break
            }
            
        })


    //  define the table that is going to be returned

    const functionMap = traces.get(traceId)?.edges

    return(
        <div>
            <p> WS status = {status} </p>
            <table>
                <thead>
                    <tr><th>FunctionFather</th><th>FunctionChildren</th></tr>
                </thead>
                <tbody>
                    {Array.from(functionMap?.entries() ?? []).map(([father,children]) => {
                        const latest = children[children.length - 1]
                        return(
                            <tr key = {father} className={latest}>
                            <td>{father}</td>
                            <td>{latest}</td>
                        </tr>
                        )
                })}
                </tbody>
            </table>
        </div>
    )


    }