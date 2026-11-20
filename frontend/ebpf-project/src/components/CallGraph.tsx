import { useEffect, useMemo, useRef } from 'react'
import * as d3 from 'd3'
import type { GraphState } from '../state/graphReduce'
import { toGraphNodes, toGraphLinks, type GraphNode, type GraphLink } from '../lib/graph'
import './CallGraph.css'

type SimNode = GraphNode & d3.SimulationNodeDatum
type SimLink = d3.SimulationLinkDatum<SimNode> & { id: string }

type D3State = {
    simulation: d3.Simulation<SimNode, SimLink>
    nodesById: Map<string, SimNode>
    nodesArr: SimNode[]
    linksById: Map<string, SimLink>
    linksArr: SimLink[]
    groupLayer: d3.Selection<SVGGElement, unknown, null, undefined>
    nodeLayer: d3.Selection<SVGGElement, unknown, null, undefined>
    edgeLayer: d3.Selection<SVGGElement, unknown, null, undefined>
}

type ProcessCircle = { x: number; y: number; r: number }

function statusVar(status: GraphNode['status']): string {
    switch (status) {
        case 'ok': return 'var(--ok)'
        case 'baseline_set': return 'var(--warn)'
        case 'regression': return 'var(--critical)'
        default: return 'var(--border)'
    }
}

function shortName(name: string): string {
    return name.replace(/^main\./, '')
}

const NODE_RADIUS = 16
// room for the node label (dy 28) and a bit of air inside the process circle
const PROCESS_PADDING = 40

function centroids(nodes: SimNode[]): Map<number, { x: number; y: number }> {
    const sums = new Map<number, { x: number; y: number; n: number }>()
    for (const d of nodes) {
        const s = sums.get(d.pid) ?? { x: 0, y: 0, n: 0 }
        s.x += d.x ?? 0
        s.y += d.y ?? 0
        s.n += 1
        sums.set(d.pid, s)
    }
    const centers = new Map<number, { x: number; y: number }>()
    for (const [pid, s] of sums) centers.set(pid, { x: s.x / s.n, y: s.y / s.n })
    return centers
}

// smallest circle around the centroid that contains every node of the process
function processCircles(nodes: SimNode[]): Map<number, ProcessCircle> {
    const centers = centroids(nodes)
    const circles = new Map<number, ProcessCircle>()
    for (const [pid, c] of centers) circles.set(pid, { x: c.x, y: c.y, r: 0 })
    for (const d of nodes) {
        const c = circles.get(d.pid)!
        const dist = Math.hypot((d.x ?? 0) - c.x, (d.y ?? 0) - c.y)
        c.r = Math.max(c.r, dist)
    }
    for (const c of circles.values()) c.r += NODE_RADIUS + PROCESS_PADDING
    return circles
}

// pulls every node toward the centroid of its process: threads of one process
// stay inside one circle, different processes drift apart (charge does the rest)
function clusterForce(strength: number) {
    let nodes: SimNode[] = []
    function force(alpha: number) {
        const centers = centroids(nodes)
        for (const d of nodes) {
            const c = centers.get(d.pid)!
            d.vx = (d.vx ?? 0) - ((d.x ?? 0) - c.x) * strength * alpha
            d.vy = (d.vy ?? 0) - ((d.y ?? 0) - c.y) * strength * alpha
        }
    }
    force.initialize = (ns: SimNode[]) => { nodes = ns }
    return force
}

export function CallGraph({ processes }: { processes: Map<number, GraphState> }) {
    const svgRef = useRef<SVGSVGElement | null>(null)
    const wrapRef = useRef<HTMLDivElement | null>(null)
    const d3Ref = useRef<D3State | null>(null)

    const nodes = useMemo(
        () => Array.from(processes).flatMap(([pid, graph]) => toGraphNodes(graph, pid)),
        [processes]
    )
    const links = useMemo(
        () => Array.from(processes).flatMap(([pid, graph]) => toGraphLinks(graph, pid)),
        [processes]
    )

    // mount: create the simulation and DOM layers once
    useEffect(() => {
        const svg = d3.select(svgRef.current!)

        // flecha para las aristas -- se referencia desde CallGraph.css (marker-end)
        svg.append('defs').append('marker')
            .attr('id', 'callgraph-arrow')
            .attr('viewBox', '0 -5 10 10')
            .attr('refX', 9)
            .attr('refY', 0)
            .attr('markerWidth', 7)
            .attr('markerHeight', 7)
            .attr('orient', 'auto')
            .append('path')
            .attr('d', 'M0,-5 L10,0 L0,5')
            .attr('fill', 'var(--edge)')

        const zoomLayer = svg.append('g').attr('class', 'zoom-layer')
        // first child = drawn underneath edges and nodes
        const groupLayer = zoomLayer.append('g').attr('class', 'group-layer')
        const edgeLayer = zoomLayer.append('g').attr('class', 'edge-layer')
        const nodeLayer = zoomLayer.append('g').attr('class', 'node-layer')

        svg.call(
            d3.zoom<SVGSVGElement, unknown>()
                .scaleExtent([0.3, 3])
                .on('zoom', (ev: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
                    zoomLayer.attr('transform', ev.transform.toString())
                })
        )

        const simulation = d3.forceSimulation<SimNode>([])
            .force('charge', d3.forceManyBody().strength(-420))
            .force('link', d3.forceLink<SimNode, SimLink>([]).id(d => d.id).distance(110).strength(0.5))
            .force('center', d3.forceCenter(0, 0))
            .force('collide', d3.forceCollide(42))
            .force('cluster', clusterForce(0.15))
            .on('tick', ticked)

        const state: D3State = {
            simulation,
            nodesById: new Map(),
            nodesArr: [],
            linksById: new Map(),
            linksArr: [],
            groupLayer,
            nodeLayer,
            edgeLayer
        }
        d3Ref.current = state

        function ticked() {
            state.edgeLayer.selectAll<SVGPathElement, SimLink>('.edge-line')
                .attr('d', d => {
                    const s = d.source as SimNode
                    const t = d.target as SimNode
                    const sx = s.x ?? 0, sy = s.y ?? 0
                    const tx = t.x ?? 0, ty = t.y ?? 0
                    const dx = tx - sx, dy = ty - sy
                    const dist = Math.sqrt(dx * dx + dy * dy) || 1
                    // frenar la línea antes del círculo del nodo destino, para
                    // que la punta de flecha se vea y no quede tapada
                    const endX = tx - (dx / dist) * (NODE_RADIUS + 3)
                    const endY = ty - (dy / dist) * (NODE_RADIUS + 3)
                    return `M${sx},${sy} L${endX},${endY}`
                })
            state.nodeLayer.selectAll<SVGGElement, SimNode>('.node-group')
                .attr('transform', d => `translate(${d.x ?? 0},${d.y ?? 0})`)

            const circles = processCircles(state.nodesArr)
            const groups = state.groupLayer.selectAll<SVGGElement, number>('.process-group')
            groups.select<SVGCircleElement>('.process-circle')
                .attr('cx', pid => circles.get(pid)?.x ?? 0)
                .attr('cy', pid => circles.get(pid)?.y ?? 0)
                .attr('r', pid => circles.get(pid)?.r ?? 0)
            groups.select<SVGTextElement>('.process-label')
                .attr('x', pid => circles.get(pid)?.x ?? 0)
                .attr('y', pid => {
                    const c = circles.get(pid)
                    return c ? c.y - c.r - 6 : 0
                })
        }

        function resize() {
            const wrap = wrapRef.current
            if (!wrap) return
            const width = wrap.clientWidth
            const height = wrap.clientHeight
            svg.attr('viewBox', `${-width / 2} ${-height / 2} ${width} ${height}`)
        }
        resize()
        const observer = new ResizeObserver(resize)
        if (wrapRef.current) observer.observe(wrapRef.current)

        return () => {
            observer.disconnect()
            simulation.stop()
            svg.selectAll('*').remove()
            d3Ref.current = null
        }
    }, [])

    // data: merge derived nodes/links into the persistent D3 state, mutating
    // in place so existing nodes keep their x/y/vx/vy across updates
    useEffect(() => {
        const state = d3Ref.current
        if (!state) return

        let structureChanged = false

        for (const n of nodes) {
            const existing = state.nodesById.get(n.id)
            if (existing) {
                existing.status = n.status
                existing.duration = n.duration
                existing.driftPct = n.driftPct
                existing.callCount = n.callCount
            } else {
                const fresh: SimNode = {
                    ...n,
                    x: (Math.random() - 0.5) * 60,
                    y: (Math.random() - 0.5) * 60
                }
                state.nodesById.set(n.id, fresh)
                state.nodesArr.push(fresh)
                structureChanged = true
            }
        }

        for (const l of links) {
            if (!state.linksById.has(l.id)) {
                const fresh: SimLink = { id: l.id, source: l.source, target: l.target }
                state.linksById.set(l.id, fresh)
                state.linksArr.push(fresh)
                structureChanged = true
            }
        }

        const edgeSel = state.edgeLayer.selectAll<SVGPathElement, SimLink>('.edge-line')
            .data(state.linksArr, (d) => (d as SimLink).id)
        edgeSel.enter().append('path').attr('class', 'edge-line')

        const nodeSel = state.nodeLayer.selectAll<SVGGElement, SimNode>('.node-group')
            .data(state.nodesArr, (d) => (d as SimNode).id)
        const nodeEnter = nodeSel.enter().append('g')
            .attr('class', 'node-group')
            .call(
                d3.drag<SVGGElement, SimNode>()
                    .on('start', (ev, d) => { if (!ev.active) state.simulation.alphaTarget(0.25).restart(); d.fx = d.x; d.fy = d.y })
                    .on('drag', (ev, d) => { d.fx = ev.x; d.fy = ev.y })
                    .on('end', (ev, d) => { if (!ev.active) state.simulation.alphaTarget(0); d.fx = null; d.fy = null })
            )
        nodeEnter.append('circle').attr('class', 'node-circle').attr('r', 16)
        nodeEnter.append('text').attr('class', 'node-label').attr('dy', 28)

        const nodeMerged = nodeEnter.merge(nodeSel)
        nodeMerged.select<SVGCircleElement>('.node-circle').attr('stroke', (d) => statusVar(d.status))
        nodeMerged.select<SVGTextElement>('.node-label').text((d) => shortName(d.name))

        const pids = Array.from(new Set(state.nodesArr.map((d) => d.pid)))
        const groupSel = state.groupLayer.selectAll<SVGGElement, number>('.process-group')
            .data(pids, (d) => d as number)
        const groupEnter = groupSel.enter().append('g').attr('class', 'process-group')
        groupEnter.append('circle').attr('class', 'process-circle')
        groupEnter.append('text').attr('class', 'process-label').text((pid) => `pid ${pid}`)
        groupSel.exit().remove()

        if (structureChanged) {
            state.simulation.nodes(state.nodesArr)
            const linkForce = state.simulation.force('link') as d3.ForceLink<SimNode, SimLink>
            linkForce.links(state.linksArr)
            state.simulation.alpha(0.5).restart()
        }
    }, [nodes, links])

    return (
        <div className="callgraph-wrap" ref={wrapRef}>
            <svg ref={svgRef} className="callgraph-svg" />
        </div>
    )
}
