import { api } from './client'

/**
 * Network interface as returned by the server.
 */
export interface NetworkInterface {
  name: string
  ip_address: string
  subnet: string
  mac: string
  status: 'up' | 'down'
}

/**
 * Scan interface setting response.
 */
export interface ScanInterfaceSetting {
  interface_name: string
}

/**
 * Fetch all available network interfaces.
 */
export async function getNetworkInterfaces(): Promise<NetworkInterface[]> {
  return api.get<NetworkInterface[]>('/settings/interfaces')
}

/**
 * Get the currently configured scan interface.
 */
export async function getScanInterface(): Promise<ScanInterfaceSetting> {
  return api.get<ScanInterfaceSetting>('/settings/scan-interface')
}

/**
 * Set the preferred scan interface.
 * @param interfaceName The interface name, or empty string for auto-detect
 */
export async function setScanInterface(interfaceName: string): Promise<ScanInterfaceSetting> {
  return api.post<ScanInterfaceSetting>('/settings/scan-interface', {
    interface_name: interfaceName,
  })
}
