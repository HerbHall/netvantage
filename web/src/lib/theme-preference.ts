/**
 * OS color scheme preference detection.
 * Used to auto-select a matching theme during setup and as the store default.
 */

/**
 * Detects the user's OS color scheme preference.
 * Returns 'dark' or 'light'. Defaults to 'dark' in non-browser environments.
 */
export function detectColorScheme(): 'dark' | 'light' {
  if (typeof window === 'undefined') return 'dark'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

/**
 * Returns the default built-in theme ID based on OS preference.
 */
export function getDefaultThemeId(): string {
  return detectColorScheme() === 'dark' ? 'builtin-forest-dark' : 'builtin-forest-light'
}
