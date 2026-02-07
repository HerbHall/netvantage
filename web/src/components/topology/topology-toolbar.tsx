import { memo, useState, type RefObject } from 'react'
import { useReactFlow } from '@xyflow/react'
import { toPng } from 'html-to-image'
import { toast } from 'sonner'
import {
  ZoomIn,
  ZoomOut,
  Maximize,
  Map,
  CircleDot,
  GitBranch,
  LayoutGrid,
  Download,
} from 'lucide-react'

export type LayoutAlgorithm = 'circular' | 'hierarchical' | 'grid'

interface TopologyToolbarProps {
  layout: LayoutAlgorithm
  onLayoutChange: (layout: LayoutAlgorithm) => void
  showMinimap: boolean
  onMinimapToggle: () => void
  flowRef: RefObject<HTMLDivElement | null>
}

const layoutOptions: { value: LayoutAlgorithm; label: string; Icon: typeof CircleDot }[] = [
  { value: 'circular', label: 'Circular', Icon: CircleDot },
  { value: 'hierarchical', label: 'Hierarchical', Icon: GitBranch },
  { value: 'grid', label: 'Grid', Icon: LayoutGrid },
]

export const TopologyToolbar = memo(function TopologyToolbar({
  layout,
  onLayoutChange,
  showMinimap,
  onMinimapToggle,
  flowRef,
}: TopologyToolbarProps) {
  const { zoomIn, zoomOut, fitView } = useReactFlow()
  const [exporting, setExporting] = useState(false)

  const handleExport = async () => {
    const element = flowRef.current
    if (!element) return

    setExporting(true)
    try {
      const dataUrl = await toPng(element, {
        backgroundColor: '#1a1a2e',
        quality: 1,
        pixelRatio: 2,
      })
      const link = document.createElement('a')
      const date = new Date().toISOString().split('T')[0]
      link.download = `subnetree-topology-${date}.png`
      link.href = dataUrl
      link.click()
      toast.success('Topology exported as PNG')
    } catch {
      toast.error('Failed to export topology')
    } finally {
      setExporting(false)
    }
  }

  return (
    <div
      className="flex items-center gap-1 rounded-lg px-2 py-1.5 shadow-md"
      style={{
        backgroundColor: 'var(--nv-bg-card)',
        border: '1px solid var(--nv-border-default)',
        backdropFilter: 'blur(8px)',
      }}
    >
      {/* Layout selector */}
      {layoutOptions.map(({ value, label, Icon }) => (
        <button
          key={value}
          onClick={() => onLayoutChange(value)}
          title={`${label} layout`}
          className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-xs font-medium transition-colors"
          style={{
            backgroundColor: layout === value ? 'var(--nv-bg-active)' : 'transparent',
            color: layout === value ? 'var(--nv-text-accent)' : 'var(--nv-text-secondary)',
          }}
        >
          <Icon className="h-3.5 w-3.5" />
          <span className="hidden sm:inline">{label}</span>
        </button>
      ))}

      {/* Separator */}
      <div
        className="h-5 w-px mx-1"
        style={{ backgroundColor: 'var(--nv-border-default)' }}
      />

      {/* Zoom controls */}
      <button
        onClick={() => zoomIn({ duration: 200 })}
        title="Zoom in"
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        <ZoomIn className="h-4 w-4" />
      </button>
      <button
        onClick={() => zoomOut({ duration: 200 })}
        title="Zoom out"
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        <ZoomOut className="h-4 w-4" />
      </button>
      <button
        onClick={() => fitView({ duration: 300, padding: 0.2 })}
        title="Fit to view"
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        <Maximize className="h-4 w-4" />
      </button>

      {/* Separator */}
      <div
        className="h-5 w-px mx-1"
        style={{ backgroundColor: 'var(--nv-border-default)' }}
      />

      {/* Minimap toggle */}
      <button
        onClick={onMinimapToggle}
        title={showMinimap ? 'Hide minimap' : 'Show minimap'}
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{
          color: showMinimap ? 'var(--nv-text-accent)' : 'var(--nv-text-secondary)',
        }}
      >
        <Map className="h-4 w-4" />
      </button>

      {/* Separator */}
      <div
        className="h-5 w-px mx-1"
        style={{ backgroundColor: 'var(--nv-border-default)' }}
      />

      {/* Export PNG */}
      <button
        onClick={handleExport}
        disabled={exporting}
        title="Export as PNG"
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)] disabled:opacity-50"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        <Download className="h-4 w-4" />
      </button>
    </div>
  )
})
