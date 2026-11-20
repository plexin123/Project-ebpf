import { useEffect, useMemo, useRef, useState } from 'react'
import * as d3 from 'd3'
import dagre from '@dagrejs/dagre'
import type { FlowEdge, FlowNode, ProcessGraph } from '../lib/graph'
import type { FunctionEvent } from '../types/events'
import { ConvertIntoMilliSeconds } from '../lib/format'
import './CallGraph.css'

// dagre places the cards left -> right in columns (caller before callee),
// React draws the SVG, and d3.zoom only handles pan/zoom
const CARD_W = 170
const CARD_H = 46
const STRIPE_W = 4
const HEADER_H = 34      // process title
const BOX_PAD = 16
const BOX_GAP = 20       // space between process boxes
const MAX_NAME = 20      // longer names are cut with … (full name in the tooltip)
const MAX_HEIGHT = 720   // taller content is reached by panning

type PlacedCard = { x: number; y: number; node: FlowNode }
type PlacedEdge = { key: string; edge: FlowEdge; d: string }
type PlacedProcess = { graph: ProcessGraph; y: number; width: number; height: number; cards: PlacedCard[]; edges: PlacedEdge[] }
type Hovered = { pid: number; name: string } | null

function statusVar(status: FunctionEvent['status'] | 'unknown'): string {
    switch (status) {
        case 'ok': return 'var(--ok)'
        case 'baseline_set': return 'var(--warn)'
        case 'regression': return 'var(--critical)'
        default: return 'var(--border)'
    }
}

// orders::main::{{closure}} -> main::closure, main.handleA -> handleA
function shortName(name: string): string {
    const short = name
        .replace(/^main\./, '')
        .replace(/^[^:]+::/, '')
        .replace(/\{\{closure\}\}/g, 'closure')
    return short.length > MAX_NAME ? short.slice(0, MAX_NAME - 1) + '…' : short
}

function tooltip(node: FlowNode): string {
    const latest = node.latest
    if (!latest) return `${node.name}\nrunning…`
    const lines = [node.name, `total ${ConvertIntoMilliSeconds(latest.duration)}`]
    if (latest.selfTime != null) lines.push(`self  ${ConvertIntoMilliSeconds(latest.selfTime)}`)
    lines.push(`status ${latest.status}`)
    if (node.recursive) lines.push('calls itself (recursive)')
    return lines.join('\n')
}

const curve = d3.line<{ x: number; y: number }>().x((p) => p.x).y((p) => p.y).curve(d3.curveBasis)

function layoutProcess(graph: ProcessGraph, top: number): PlacedProcess {
    const g = new dagre.graphlib.Graph()
    g.setGraph({ rankdir: 'LR', nodesep: 16, ranksep: 60, marginx: BOX_PAD, marginy: 8 })
    g.setDefaultEdgeLabel(() => ({}))
    for (const node of graph.nodes) g.setNode(node.name, { width: CARD_W, height: CARD_H })
    for (const edge of graph.edges) g.setEdge(edge.source, edge.target)
    dagre.layout(g)

    // dagre gives card centers; shift everything below the process title
    const cards = graph.nodes.map((node) => {
        const { x, y } = g.node(node.name)
        return { x: x - CARD_W / 2, y: y - CARD_H / 2 + HEADER_H, node }
    })
    const edges = graph.edges.map((edge) => {
        const points = g.edge(edge.source, edge.target).points.map((p) => ({ x: p.x, y: p.y + HEADER_H }))
        return { key: `${edge.source}->${edge.target}`, edge, d: curve(points) ?? '' }
    })
    const size = g.graph()
    return {
        graph,
        y: top,
        width: Math.max((size.width ?? 0), 360),
        height: (size.height ?? 0) + HEADER_H,
        cards,
        edges
    }
}

// everything that calls `name` (directly or not) and everything it calls
function relatedTo(graph: ProcessGraph, name: string) {
    const walk = (start: string, next: (n: string) => string[]) => {
        const seen = new Set<string>([start])
        const queue = [start]
        while (queue.length) {
            for (const n of next(queue.shift()!)) {
                if (!seen.has(n)) {
                    seen.add(n)
                    queue.push(n)
                }
            }
        }
        return seen
    }
    const callers = walk(name, (n) => graph.edges.filter((e) => e.target === n).map((e) => e.source))
    const callees = walk(name, (n) => graph.edges.filter((e) => e.source === n).map((e) => e.target))
    return {
        node: (n: string) => callers.has(n) || callees.has(n),
        edge: (e: FlowEdge) =>
            (callers.has(e.source) && callers.has(e.target)) || (callees.has(e.source) && callees.has(e.target))
    }
}

export function CallGraph({ processes }: { processes: ProcessGraph[] }) {
    const svgRef = useRef<SVGSVGElement | null>(null)
    const zoomRef = useRef<SVGGElement | null>(null)
    const [hovered, setHovered] = useState<Hovered>(null)

    const layout = useMemo(() => {
        let top = 0
        const placed = processes.map((graph) => {
            const p = layoutProcess(graph, top)
            top += p.height + BOX_GAP
            return p
        })
        return { processes: placed, height: Math.max(top - BOX_GAP, 0) }
    }, [processes])

    // pan/zoom only moves the content group; the layout itself never changes
    useEffect(() => {
        const svg = d3.select(svgRef.current!)
        const content = d3.select(zoomRef.current!)
        svg.call(
            d3.zoom<SVGSVGElement, unknown>()
                .scaleExtent([0.3, 3])
                .on('zoom', (ev: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
                    content.attr('transform', ev.transform.toString())
                })
        )
        return () => { svg.on('.zoom', null) }
    }, [])

    const height = Math.min(Math.max(layout.height + 2, 160), MAX_HEIGHT)

    return (
        <div className="callgraph">
            <div className="callgraph-wrap" style={{ height }}>
                {processes.length === 0 && <div className="callgraph-empty">waiting for events…</div>}
                <svg ref={svgRef} className="callgraph-svg">
                    <defs>
                        <marker id="callgraph-arrow" viewBox="0 -5 10 10" refX={9} refY={0}
                            markerWidth={7} markerHeight={7} orient="auto">
                            <path d="M0,-5 L10,0 L0,5" fill="var(--edge)" />
                        </marker>
                    </defs>
                    <g ref={zoomRef}>
                        {layout.processes.map((process) => {
                            const { graph } = process
                            const focus = hovered?.pid === graph.pid ? relatedTo(graph, hovered.name) : null
                            return (
                                <g key={graph.pid} transform={`translate(0,${process.y})`}>
                                    <rect className="process-box" width={process.width} height={process.height} rx={10} />
                                    <text className="process-label" x={BOX_PAD} y={22}>
                                        {graph.label}
                                        <tspan className="process-threads">
                                            {`  ·  ${graph.threadCount} thread${graph.threadCount === 1 ? '' : 's'}`}
                                        </tspan>
                                    </text>

                                    {process.edges.map(({ key, edge, d }) => (
                                        <path key={key} d={d}
                                            className={`edge-line${focus ? (focus.edge(edge) ? ' is-active' : ' is-dim') : ''}`} />
                                    ))}

                                    {process.cards.map(({ x, y, node }) => {
                                        const status = node.latest?.status ?? 'unknown'
                                        const drift = node.latest?.driftPct ?? 0
                                        const dim = focus && !focus.node(node.name)
                                        return (
                                            <g key={node.name} transform={`translate(${x},${y})`}
                                                className={`card${dim ? ' is-dim' : ''}`}
                                                onMouseEnter={() => setHovered({ pid: graph.pid, name: node.name })}
                                                onMouseLeave={() => setHovered(null)}>
                                                <title>{tooltip(node)}</title>
                                                <rect className="card-bg" width={CARD_W} height={CARD_H} rx={6} />
                                                <rect className="card-stripe" width={STRIPE_W} height={CARD_H} rx={2}
                                                    fill={statusVar(status)} />
                                                <text className="card-name" x={STRIPE_W + 10} y={19}>
                                                    {shortName(node.name)}{node.recursive ? ' ↻' : ''}
                                                </text>
                                                <text className="card-time" x={STRIPE_W + 10} y={36}>
                                                    {node.latest ? ConvertIntoMilliSeconds(node.latest.duration) : 'running…'}
                                                    {status === 'regression' && (
                                                        <tspan className="card-drift">{`  ▲ +${drift.toFixed(0)}%`}</tspan>
                                                    )}
                                                </text>
                                            </g>
                                        )
                                    })}
                                </g>
                            )
                        })}
                    </g>
                </svg>
            </div>
            <div className="callgraph-legend">
                <span><i style={{ background: 'var(--ok)' }} />ok</span>
                <span><i style={{ background: 'var(--warn)' }} />new baseline</span>
                <span><i style={{ background: 'var(--critical)' }} />regression</span>
                <span><i style={{ background: 'var(--border)' }} />running</span>
                <span className="callgraph-hint">hover a function to follow its path · drag to pan · scroll to zoom</span>
            </div>
        </div>
    )
}
