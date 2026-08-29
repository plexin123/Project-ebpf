import{
    createContext, use, useContext,useReducer, type ReactNode
} from 'react'
import {tracesReducer, initialTracesState, type TracesState, type GraphAction } from './graphReduce'

export type GraphContextValue = {
    traces: TracesState
    dispatch: React.Dispatch<GraphAction>
}
const GraphContext = createContext<GraphContextValue | null>(null)

export function GraphProvider({children}: {children: ReactNode}){
    const [traces, dispatch] = useReducer(tracesReducer, initialTracesState)

    return(
        <GraphContext.Provider value={{traces, dispatch}}>
            {children}
        </GraphContext.Provider>
    )
}

export function useGraph(): GraphContextValue{
    const context = useContext(GraphContext)
    if (!context){
        throw new Error("useGraph must be used within a GraphProvider")
    }
    return context
}