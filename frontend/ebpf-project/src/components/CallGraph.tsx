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
    nodeLayer: d3.Selection<SVGGElement, unknown, null, undefined>
    edgeLayer: d3.Selection<SVGGElement, unknown, null, undefined>
}

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

export function CallGraph({ graphState }: { graphState: GraphState | undefined }) {
    const svgRef = useRef<SVGSVGElement | null>(null)
    const wrapRef = useRef<HTMLDivElement | null>(null)
    const d3Ref = useRef<D3State | null>(null)

    const nodes = useMemo(() => toGraphNodes(graphState), [graphState])
    const links = useMemo(() => toGraphLinks(graphState), [graphState])

    // mount: create the simulation and DOM layers once
    useEffect(() => {
        const svg = d3.select(svgRef.current!)
        const zoomLayer = svg.append('g').attr('class', 'zoom-layer')
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
            .on('tick', ticked)

        const state: D3State = {
            simulation,
            nodesById: new Map(),
            nodesArr: [],
            linksById: new Map(),
            linksArr: [],
            nodeLayer,
            edgeLayer
        }
        d3Ref.current = state

        function ticked() {
            state.edgeLayer.selectAll<SVGPathElement, SimLink>('.edge-line')
                .attr('d', d => {
                    const s = d.source as SimNode
                    const t = d.target as SimNode
                    return `M${s.x ?? 0},${s.y ?? 0} L${t.x ?? 0},${t.y ?? 0}`
                })
            state.nodeLayer.selectAll<SVGGElement, SimNode>('.node-group')
                .attr('transform', d => `translate(${d.x ?? 0},${d.y ?? 0})`)
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
        nodeMerged.select<SVGTextElement>('.node-label').text((d) => shortName(d.id))

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
