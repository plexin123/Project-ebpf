import { useEffect, useMemo, useRef } from 'react'
import * as d3 from 'd3'
import type { CallTreeNode, ProcessView } from '../lib/graph'
import type { FunctionEvent } from '../types/events'
import { ConvertIntoMilliSeconds } from '../lib/format'
import './CallGraph.css'

// layout is deterministic (no physics): d3 computes the tree positions,
// React draws the SVG, and d3.zoom only handles pan/zoom
const COL_W = 200        // horizontal distance between a caller and its callee
const ROW_H = 52         // vertical space per function (name above, timings below)
const NODE_R = 7
const LANE_LABEL_W = 130 // left column with the thread name
const BOX_PAD = 16
const HEADER_H = 30      // process title
const BOX_GAP = 20       // space between process boxes
const MAX_HEIGHT = 720   // taller content is reached by panning

type PlacedNode = { key: string; x: number; y: number; node: CallTreeNode }
type PlacedLink = { key: string; d: string }
type PlacedLane = { tid: number; y: number; height: number; nodes: PlacedNode[]; links: PlacedLink[] }
type PlacedProcess = { pid: number; label: string; y: number; height: number; lanes: PlacedLane[] }
type Layout = { width: number; height: number; processes: PlacedProcess[] }

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
    return name
        .replace(/^main\./, '')
        .replace(/^[^:]+::/, '')
        .replace(/\{\{closure\}\}/g, 'closure')
}

function laneName(pid: number, tid: number): string {
    // on Linux the main thread's tid is the process id
    return tid === pid ? `main thread ${tid}` : `thread ${tid}`
}

function timings(node: CallTreeNode): string {
    const latest = node.latest
    if (!latest) return 'running…'
    const total = ConvertIntoMilliSeconds(latest.duration)
    return latest.selfTime != null ? `${total} · self ${ConvertIntoMilliSeconds(latest.selfTime)}` : total
}

const link = d3.linkHorizontal<{ source: [number, number]; target: [number, number] }, [number, number]>()

function layoutLane(pid: number, tid: number, roots: CallTreeNode[], top: number): PlacedLane {
    // one virtual root so several real roots share a single tree layout
    const virtual: CallTreeNode = { name: '', recursive: false, children: roots }
    const tree = d3.tree<CallTreeNode>().nodeSize([ROW_H, COL_W]).separation(() => 1)(d3.hierarchy(virtual))
    const real = tree.descendants().filter((d) => d.depth > 0)

    const minX = d3.min(real, (d) => d.x) ?? 0
    const maxX = d3.max(real, (d) => d.x) ?? 0
    const left = BOX_PAD + LANE_LABEL_W
    // d3.tree's x is vertical and y is horizontal; depth 1 is the first real column
    const toScreen = (d: d3.HierarchyPointNode<CallTreeNode>): [number, number] =>
        [left + d.y - COL_W, top + d.x - minX + ROW_H / 2]
    const pathKey = (d: d3.HierarchyPointNode<CallTreeNode>) =>
        `${pid}:${tid}:` + d.ancestors().map((a) => a.data.name).reverse().join('>')

    const nodes = real.map((d) => {
        const [x, y] = toScreen(d)
        return { key: pathKey(d), x, y, node: d.data }
    })
    const links = tree.links()
        .filter((l) => l.source.depth > 0)
        .map((l) => {
            const [sx, sy] = toScreen(l.source)
            const [tx, ty] = toScreen(l.target)
            // stop before the target circle so the arrow head stays visible
            const d = link({ source: [sx + NODE_R, sy], target: [tx - NODE_R - 3, ty] }) ?? ''
            return { key: pathKey(l.target), d }
        })
    return { tid, y: top, height: maxX - minX + ROW_H, nodes, links }
}

function computeLayout(processes: ProcessView[]): Layout {
    let y = 0
    let width = 0
    const placed: PlacedProcess[] = []
    for (const process of processes) {
        const lanes: PlacedLane[] = []
        let laneTop = HEADER_H
        for (const thread of process.threads) {
            const lane = layoutLane(process.pid, thread.tid, thread.roots, laneTop)
            lanes.push(lane)
            laneTop += lane.height
            for (const n of lane.nodes) width = Math.max(width, n.x + COL_W)
        }
        const height = laneTop + BOX_PAD / 2
        placed.push({ pid: process.pid, label: process.label, y, height, lanes })
        y += height + BOX_GAP
    }
    return { width: Math.max(width, 400), height: Math.max(y - BOX_GAP, 0), processes: placed }
}

export function CallGraph({ processes }: { processes: ProcessView[] }) {
    const svgRef = useRef<SVGSVGElement | null>(null)
    const zoomRef = useRef<SVGGElement | null>(null)

    const layout = useMemo(() => computeLayout(processes), [processes])

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
                    {layout.processes.map((process) => (
                        <g key={process.pid} transform={`translate(0,${process.y})`}>
                            <rect className="process-box" width={layout.width} height={process.height} rx={8} />
                            <text className="process-label" x={BOX_PAD} y={20}>{process.label}</text>

                            {process.lanes.map((lane, i) => (
                                <g key={lane.tid}>
                                    {i > 0 && (
                                        <line className="lane-divider" x1={BOX_PAD} x2={layout.width - BOX_PAD}
                                            y1={lane.y} y2={lane.y} />
                                    )}
                                    <text className="lane-label" x={BOX_PAD} y={lane.y + ROW_H / 2 + 4}>
                                        {laneName(process.pid, lane.tid)}
                                    </text>

                                    {lane.links.map((l) => <path key={l.key} className="edge-line" d={l.d} />)}

                                    {lane.nodes.map(({ key, x, y, node }) => (
                                        <g key={key} className="node-group" transform={`translate(${x},${y})`}>
                                            <title>{`${node.name}\n${timings(node)}`}</title>
                                            <circle className="node-circle" r={NODE_R}
                                                stroke={statusVar(node.latest?.status ?? 'unknown')} />
                                            <text className="node-label" x={-NODE_R} y={-NODE_R - 6}>
                                                {shortName(node.name)}{node.recursive ? ' ↻' : ''}
                                            </text>
                                            <text className="node-meta" x={-NODE_R} y={NODE_R + 14}>
                                                {timings(node)}
                                            </text>
                                        </g>
                                    ))}
                                </g>
                            ))}
                        </g>
                    ))}
                </g>
            </svg>
        </div>
    )
}
