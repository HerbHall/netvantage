import { memo, useState } from 'react'
import {
  BaseEdge,
  EdgeLabelRenderer,
  getSmoothStepPath,
  type EdgeProps,
  type Edge,
} from '@xyflow/react'

export interface TopologyEdgeData {
  linkType?: string
  [key: string]: unknown
}

export type TopologyEdgeType = Edge<TopologyEdgeData, 'topology'>

export const TopologyEdge = memo(function TopologyEdge(
  props: EdgeProps<TopologyEdgeType>
) {
  const {
    id,
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    data,
    selected,
  } = props

  const [hovered, setHovered] = useState(false)

  const [edgePath, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  })

  const strokeColor =
    selected || hovered
      ? 'var(--nv-topo-link-active)'
      : 'var(--nv-topo-link-default)'

  return (
    <>
      {/* Invisible wider path for easier hover interaction */}
      <path
        d={edgePath}
        fill="none"
        stroke="transparent"
        strokeWidth={20}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      />
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: strokeColor,
          strokeWidth: selected || hovered ? 2.5 : 1.5,
          transition: 'stroke 0.2s, stroke-width 0.2s',
        }}
      />
      {/* Animated dot for online/active edges */}
      {(selected || hovered) && (
        <circle r="3" fill="var(--nv-topo-link-active)">
          <animateMotion dur="3s" repeatCount="indefinite" path={edgePath} />
        </circle>
      )}
      {/* Link type label on hover */}
      {hovered && data?.linkType && (
        <EdgeLabelRenderer>
          <div
            className="nodrag nopan pointer-events-none rounded px-1.5 py-0.5 text-[10px] font-mono"
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              backgroundColor: 'var(--nv-bg-elevated)',
              color: 'var(--nv-text-secondary)',
              border: '1px solid var(--nv-border-default)',
            }}
          >
            {data.linkType}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
})
