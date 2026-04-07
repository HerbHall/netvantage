import { memo, useMemo } from 'react'
import {
  Search,
  RotateCcw,
  Server,
  Monitor,
  Laptop,
  Smartphone,
  Router,
  Network,
  Wifi,
  Shield,
  Printer,
  HardDrive,
  Cpu,
  Phone,
  Tablet,
  Camera,
  Container,
  CircleHelp,
  type LucideIcon,
} from 'lucide-react'
import type { DeviceStatus, DeviceType, TopologyNode } from '@/api/types'

interface TopologyFiltersProps {
  nodes: TopologyNode[]
  statusFilter: Set<DeviceStatus>
  onStatusFilterChange: (status: DeviceStatus) => void
  typeFilter: Set<DeviceType>
  onTypeFilterChange: (type: DeviceType) => void
  searchQuery: string
  onSearchChange: (query: string) => void
  onReset: () => void
  visibleCount: number
}

const statusOptions: { value: DeviceStatus; label: string; color: string }[] = [
  { value: 'online', label: 'Online', color: 'var(--nv-status-online)' },
  { value: 'offline', label: 'Offline', color: 'var(--nv-status-offline)' },
  { value: 'degraded', label: 'Degraded', color: 'var(--nv-status-degraded)' },
  { value: 'unknown', label: 'Unknown', color: 'var(--nv-status-unknown)' },
]

const deviceTypeIcons: Record<DeviceType, LucideIcon> = {
  server: Server,
  desktop: Monitor,
  laptop: Laptop,
  mobile: Smartphone,
  router: Router,
  switch: Network,
  access_point: Wifi,
  firewall: Shield,
  printer: Printer,
  nas: HardDrive,
  iot: Cpu,
  phone: Phone,
  tablet: Tablet,
  camera: Camera,
  virtual_machine: Server,
  container: Container,
  unknown: CircleHelp,
}

const deviceTypeLabels: Record<DeviceType, string> = {
  server: 'Server',
  desktop: 'Desktop',
  laptop: 'Laptop',
  mobile: 'Mobile',
  router: 'Router',
  switch: 'Switch',
  access_point: 'AP',
  firewall: 'Firewall',
  printer: 'Printer',
  nas: 'NAS',
  iot: 'IoT',
  phone: 'Phone',
  tablet: 'Tablet',
  camera: 'Camera',
  virtual_machine: 'VM',
  container: 'Container',
  unknown: 'Unknown',
}

export const TopologyFilters = memo(function TopologyFilters({
  nodes,
  statusFilter,
  onStatusFilterChange,
  typeFilter,
  onTypeFilterChange,
  searchQuery,
  onSearchChange,
  onReset,
  visibleCount,
}: TopologyFiltersProps) {
  // Only show device types that exist in the data
  const presentTypes = useMemo(() => {
    const types = new Set<DeviceType>()
    for (const n of nodes) {
      types.add(n.device_type)
    }
    return Array.from(types).sort()
  }, [nodes])

  const totalCount = nodes.length
  const hasActiveFilters =
    statusFilter.size < 4 || typeFilter.size < presentTypes.length || searchQuery.length > 0

  return (
    <>
      {/* Search */}
      <div
        className="flex items-center gap-1.5 rounded-md px-2 py-1 w-40"
        style={{
          backgroundColor: 'var(--nv-input-bg)',
          border: '1px solid var(--nv-input-border)',
        }}
      >
        <Search
          className="h-3 w-3 flex-shrink-0"
          style={{ color: 'var(--nv-text-secondary)' }}
        />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search..."
          className="bg-transparent text-[11px] outline-none flex-1 min-w-0"
          style={{ color: 'var(--nv-input-text)' }}
        />
      </div>

      {/* Status pills */}
      {statusOptions.map(({ value, label, color }) => (
        <button
          key={value}
          onClick={() => onStatusFilterChange(value)}
          title={label}
          className="flex items-center gap-1 rounded-md px-1.5 py-1 text-[10px] font-medium transition-colors"
          style={{
            backgroundColor: statusFilter.has(value) ? 'var(--nv-bg-active)' : 'transparent',
            color: statusFilter.has(value) ? color : 'var(--nv-text-muted)',
            opacity: statusFilter.has(value) ? 1 : 0.6,
          }}
        >
          <span
            className="h-1.5 w-1.5 rounded-full flex-shrink-0"
            style={{ backgroundColor: color }}
          />
          {label}
        </button>
      ))}

      {/* Divider */}
      <div
        className="w-px h-5 flex-shrink-0"
        style={{ backgroundColor: 'var(--nv-border-default)' }}
      />

      {/* Type filter icons */}
      {presentTypes.map((type) => {
        const Icon = deviceTypeIcons[type] || CircleHelp
        const label = deviceTypeLabels[type] || type
        const active = typeFilter.has(type)
        return (
          <button
            key={type}
            onClick={() => onTypeFilterChange(type)}
            title={label}
            className="rounded-md p-1 transition-colors"
            style={{
              backgroundColor: active ? 'var(--nv-bg-active)' : 'transparent',
              color: active ? 'var(--nv-text-primary)' : 'var(--nv-text-muted)',
              opacity: active ? 1 : 0.4,
            }}
          >
            <Icon className="h-3.5 w-3.5" />
          </button>
        )
      })}

      {/* Count */}
      <span
        className="text-[10px] font-medium flex-shrink-0"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        {visibleCount}/{totalCount}
      </span>

      {hasActiveFilters && (
        <button
          onClick={onReset}
          title="Reset filters"
          className="rounded-md p-1 transition-colors hover:bg-[var(--nv-bg-hover)]"
          style={{ color: 'var(--nv-text-secondary)' }}
        >
          <RotateCcw className="h-3 w-3" />
        </button>
      )}
    </>
  )
})
