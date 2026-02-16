import { api } from './client'
import type { Device, SNMPSystemInfo, SNMPInterface, SNMPDiscoverRequest, TracerouteRequest, TracerouteResult } from './types'

/** Discover a device via SNMP. */
export async function discoverSNMP(req: SNMPDiscoverRequest): Promise<Device[]> {
  return api.post<Device[]>('/recon/snmp/discover', req)
}

/** Get SNMP system information for a device. */
export async function getSNMPSystemInfo(deviceId: string): Promise<SNMPSystemInfo> {
  return api.get<SNMPSystemInfo>(`/recon/snmp/system/${deviceId}`)
}

/** Get SNMP interface table for a device. */
export async function getSNMPInterfaces(deviceId: string): Promise<SNMPInterface[]> {
  return api.get<SNMPInterface[]>(`/recon/snmp/interfaces/${deviceId}`)
}

/** Run an ICMP traceroute to a target IP. */
export async function runTraceroute(req: TracerouteRequest): Promise<TracerouteResult> {
  return api.post<TracerouteResult>('/recon/traceroute', req)
}
