import ELK, { type ElkNode } from 'elkjs/lib/elk.bundled.js'

import type { DeviceNodeType } from './device-node'

export type ElkDirection = 'DOWN' | 'RIGHT'

const elk = new ELK()

const NODE_WIDTH = 180
const NODE_HEIGHT = 60

/**
 * Compute a layered layout for topology nodes using elkjs (Eclipse Layout Kernel).
 * Returns a new array of nodes with updated positions.
 */
export async function elkLayout(
  nodes: DeviceNodeType[],
  edges: { id: string; source: string; target: string }[],
  direction: ElkDirection = 'DOWN'
): Promise<DeviceNodeType[]> {
  if (nodes.length === 0) return []

  const graph: ElkNode = {
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': direction,
      'elk.spacing.nodeNode': '80',
      'elk.layered.spacing.nodeNodeBetweenLayers': '150',
      'elk.edgeRouting': 'ORTHOGONAL',
    },
    children: nodes.map((n) => ({
      id: n.id,
      width: NODE_WIDTH,
      height: NODE_HEIGHT,
    })),
    edges: edges.map((e) => ({
      id: e.id,
      sources: [e.source],
      targets: [e.target],
    })),
  }

  const layout = await elk.layout(graph)

  const positionMap = new Map<string, { x: number; y: number }>()
  for (const child of layout.children ?? []) {
    positionMap.set(child.id, { x: child.x ?? 0, y: child.y ?? 0 })
  }

  return nodes.map((node) => ({
    ...node,
    position: positionMap.get(node.id) ?? node.position,
  }))
}
