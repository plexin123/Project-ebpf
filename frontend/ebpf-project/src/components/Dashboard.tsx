// showcase the graph first in a table and then a graph node


// this is going to connect to the backend using websockets
// then create a graph using the data received from the backend
// then display the graph in a table and then a graph node

import React, { useEffect, useState } from 'react'

import { WebsocketConnectionEntrance } from '../hooks/useWebsockets'
import { useGraph } from '../state/GraphContext'
import { CallEvent, FunctionEvent, WSMessage } from '../types/events'
import { tracesReducer } from '../state/graphReduce'
import { GraphAction } from '../state/graphReduce'
import { GraphState } from '../state/graphReduce'


const DEFAULT_URL = "None"

function Dashboard() {
    const {traces,  dispatch} = useGraph()
    WebsocketConnectionEntrance(DEFAULT_URL,(ws_message : WSMessage | null) =>{            
        
        switch(ws_message?.type){
            case("connection"):
                let callEvent: CallEvent = ws_message.payload as CallEvent
                dispatch({type: 'ADD_EDGE', payload: callEvent, traceId: ws_message.traceId })
            
            case("event"):
                let functionEvent : FunctionEvent = ws_message.payload as FunctionEvent
                dispatch({type: 'UPSERT_NODE' , payload: functionEvent , traceId: ws_message.traceId })
            
        }
    })
}