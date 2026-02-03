/** JWT access + refresh token pair returned by login and refresh endpoints. */
export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_in: number
}

/** User account as returned by the server. */
export interface User {
  id: string
  username: string
  email: string
  role: 'admin' | 'operator' | 'viewer'
  auth_provider: string
  oidc_subject?: string
  created_at: string
  last_login?: string
  disabled: boolean
  locked_until?: string
}

/** RFC 7807 Problem Detail error response. */
export interface ProblemDetail {
  type: string
  title: string
  status: number
  detail?: string
  instance?: string
}
