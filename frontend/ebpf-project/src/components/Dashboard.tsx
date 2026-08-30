// showcase the graph first in a table and then a graph node

// this is going to connect to the backend using websockets
// then create a graph using the data received from the backend
// then display the graph in a table and then a graph node

import { useMemo, useState } from 'react'

import { WebsocketConnectionEntrance } from '../hooks/useWebsockets'
import { useGraph } from '../state/GraphContext'
import { CallEvent, FunctionEvent, WSMessage } from '../types/events'
import { mergeTraces } from '../lib/graph'
import { CallGraph } from './CallGraph'
import './Dashboard.css'

const DEFAULT_URL = ""

export function Dashboard() {
    const { traces, dispatch } = useGraph()
    const [traceId, setTraceId] = useState("")
    const status = WebsocketConnectionEntrance(DEFAULT_URL, (ws_message: WSMessage) => {
        setTraceId(ws_message.traceId)
        console.log(ws_message)
        switch (ws_message?.type) {
            case 'connection': {
                let callEvent: CallEvent = ws_message.payload as CallEvent
                console.log(callEvent)
                dispatch({ type: 'ADD_EDGE', payload: callEvent, traceId: ws_message.traceId })
                break
            }
            case 'event': {
                let functionEvent: FunctionEvent = ws_message.payload as FunctionEvent
                dispatch({ type: 'UPSERT_NODE', payload: functionEvent, traceId: ws_message.traceId })
                break
            }
        }
    })

    // se usa el merge de todos los traces (no solo el traceId más reciente) porque
    // el backend genera un traceId nuevo por cada llamada de nivel superior — con
    // un binario que hace muchas llamadas cortas, leer solo el último dejaría el
    // grafo "reiniciándose" todo el tiempo
    const merged = useMemo(() => mergeTraces(traces), [traces])
    const edges = merged.edges
    const functionStats = merged.node

    return (
        <div className="dashboard">
            <header className="dashboard-header">
                <h1>eBPF Trace Monitor</h1>
                <span className={`status-pill status-${status}`}>{status}</span>
            </header>

            <section className="panel">
                <h2>Grafo</h2>
                <CallGraph graphState={merged} />
            </section>

            <section className="panel">
                <h2>Funciones</h2>
                <table>
                    <thead>
                        <tr><th>Función</th><th>Status</th><th>Duración</th><th>Drift</th></tr>
                    </thead>
                    <tbody>
                        {Array.from(functionStats?.entries() ?? []).map(([funcName, history]) => {
                            const latest = history[history.length - 1]
                            return (
                                <tr key={funcName}>
                                    <td className="mono">{funcName}</td>
                                    <td><span className={`badge badge-${latest.status}`}>{latest.status}</span></td>
                                    <td className="num">{latest.duration}</td>
                                    <td className="num">{latest.driftPct ? `${latest.driftPct.toFixed(1)}%` : '—'}</td>
                                </tr>
                            )
                        })}
                        {(!functionStats || functionStats.size === 0) && (
                            <tr className="empty-row"><td colSpan={4}>esperando eventos…</td></tr>
                        )}
                    </tbody>
                </table>
            </section>

            <section className="panel">
                <h2>Relaciones</h2>
                <table>
                    <thead>
                        <tr><th>Función</th><th>Llama a</th></tr>
                    </thead>
                    <tbody>
                        {Array.from(edges?.entries() ?? []).map(([father, children]) => (
                            <tr key={father}>
                                <td className="mono">{father}</td>
                                <td className="mono">{children.join(', ')}</td>
                            </tr>
                        ))}
                        {(!edges || edges.size === 0) && (
                            <tr className="empty-row"><td colSpan={2}>esperando eventos…</td></tr>
                        )}
                    </tbody>
                </table>
            </section>
        </div>
    )
}
