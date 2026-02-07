import { memo, useEffect } from 'react'
import { ArrowRight } from 'lucide-react'

interface EdgeInfoData {
  id: string
  sourceLabel: string
  targetLabel: string
  linkType: string
  speed?: number
}

interface EdgeInfoPopoverProps {
  edge: EdgeInfoData
  position: { x: number; y: number }
  onDismiss: () => void
}

export const EdgeInfoPopover = memo(function EdgeInfoPopover({
  edge,
  position,
  onDismiss,
}: EdgeInfoPopoverProps) {
  // Auto-dismiss after 5 seconds
  useEffect(() => {
    const timer = setTimeout(onDismiss, 5000)
    return () => clearTimeout(timer)
  }, [edge.id, onDismiss])

  return (
    <div
      className="fixed z-50 rounded-lg px-3 py-2 shadow-lg"
      style={{
        left: position.x,
        top: position.y,
        transform: 'translate(-50%, -100%) translateY(-8px)',
        backgroundColor: 'var(--nv-bg-elevated)',
        border: '1px solid var(--nv-border-default)',
        animation: 'fadeInUp 0.15s ease-out',
      }}
    >
      {/* Source -> Target */}
      <div className="flex items-center gap-2 mb-1.5">
        <span
          className="text-xs font-medium truncate max-w-[100px]"
          style={{ color: 'var(--nv-text-primary)' }}
        >
          {edge.sourceLabel}
        </span>
        <ArrowRight
          className="h-3 w-3 flex-shrink-0"
          style={{ color: 'var(--nv-text-secondary)' }}
        />
        <span
          className="text-xs font-medium truncate max-w-[100px]"
          style={{ color: 'var(--nv-text-primary)' }}
        >
          {edge.targetLabel}
        </span>
      </div>

      {/* Link details */}
      <div className="flex items-center gap-3">
        {edge.linkType && (
          <div className="flex items-center gap-1">
            <span
              className="text-[10px] uppercase tracking-wider"
              style={{ color: 'var(--nv-text-muted)' }}
            >
              Type:
            </span>
            <span
              className="text-[10px] font-mono"
              style={{ color: 'var(--nv-text-secondary)' }}
            >
              {edge.linkType}
            </span>
          </div>
        )}
        {edge.speed != null && (
          <div className="flex items-center gap-1">
            <span
              className="text-[10px] uppercase tracking-wider"
              style={{ color: 'var(--nv-text-muted)' }}
            >
              Speed:
            </span>
            <span
              className="text-[10px] font-mono"
              style={{ color: 'var(--nv-text-secondary)' }}
            >
              {formatSpeed(edge.speed)}
            </span>
          </div>
        )}
      </div>

      {/* Animation */}
      <style>{`
        @keyframes fadeInUp {
          from {
            opacity: 0;
            transform: translate(-50%, -100%) translateY(0px);
          }
          to {
            opacity: 1;
            transform: translate(-50%, -100%) translateY(-8px);
          }
        }
      `}</style>
    </div>
  )
})

function formatSpeed(speedMbps: number): string {
  if (speedMbps >= 1000) {
    return `${(speedMbps / 1000).toFixed(speedMbps % 1000 === 0 ? 0 : 1)} Gbps`
  }
  return `${speedMbps} Mbps`
}
