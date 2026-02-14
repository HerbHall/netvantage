import { api } from './client'
import type {
  Check,
  CheckResult,
  Alert,
  CreateCheckRequest,
  UpdateCheckRequest,
  MonitoringStatus,
} from './types'

/**
 * List all monitoring checks (enabled and disabled).
 */
export async function listChecks(): Promise<Check[]> {
  return api.get<Check[]>('/pulse/checks')
}

/**
 * Get monitoring checks for a specific device.
 */
export async function getDeviceChecks(deviceId: string): Promise<Check | null> {
  return api.get<Check | null>(`/pulse/checks/${deviceId}`)
}

/**
 * Create a new monitoring check.
 */
export async function createCheck(req: CreateCheckRequest): Promise<Check> {
  return api.post<Check>('/pulse/checks', req)
}

/**
 * Update an existing monitoring check.
 */
export async function updateCheck(
  id: string,
  req: UpdateCheckRequest
): Promise<Check> {
  return api.put<Check>(`/pulse/checks/${id}`, req)
}

/**
 * Delete a monitoring check and its results.
 */
export async function deleteCheck(id: string): Promise<void> {
  return api.delete<void>(`/pulse/checks/${id}`)
}

/**
 * Toggle a check's enabled/disabled state.
 */
export async function toggleCheck(id: string): Promise<Check> {
  return api.patch<Check>(`/pulse/checks/${id}/toggle`, {})
}

/**
 * Get recent check results for a device.
 */
export async function getDeviceResults(
  deviceId: string,
  limit?: number
): Promise<CheckResult[]> {
  const query = new URLSearchParams()
  if (limit) query.set('limit', limit.toString())
  const qs = query.toString()
  return api.get<CheckResult[]>(`/pulse/results/${deviceId}${qs ? `?${qs}` : ''}`)
}

/**
 * List alerts with optional filtering.
 */
export async function listAlerts(params?: {
  device_id?: string
  severity?: string
  active?: boolean
  limit?: number
}): Promise<Alert[]> {
  const query = new URLSearchParams()
  if (params?.device_id) query.set('device_id', params.device_id)
  if (params?.severity) query.set('severity', params.severity)
  if (params?.active !== undefined) query.set('active', params.active.toString())
  if (params?.limit) query.set('limit', params.limit.toString())
  const qs = query.toString()
  return api.get<Alert[]>(`/pulse/alerts${qs ? `?${qs}` : ''}`)
}

/**
 * Get a single alert by ID.
 */
export async function getAlert(id: string): Promise<Alert> {
  return api.get<Alert>(`/pulse/alerts/${id}`)
}

/**
 * Acknowledge an alert.
 */
export async function acknowledgeAlert(id: string): Promise<Alert> {
  return api.post<Alert>(`/pulse/alerts/${id}/acknowledge`, {})
}

/**
 * Resolve an alert.
 */
export async function resolveAlert(id: string): Promise<Alert> {
  return api.post<Alert>(`/pulse/alerts/${id}/resolve`, {})
}

/**
 * Get composite monitoring status for a device.
 */
export async function getDeviceStatus(
  deviceId: string
): Promise<MonitoringStatus> {
  return api.get<MonitoringStatus>(`/pulse/status/${deviceId}`)
}
