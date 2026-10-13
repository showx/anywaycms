export function adminBase() {
  const value = typeof window === 'undefined' ? '' : window.__ADMIN_BASE__
  if (typeof value !== 'string' || !/^\/[a-z][a-z0-9]{2,31}$/.test(value) || value === '/admin') return ''
  return value
}

export function adminPath(suffix = '') {
  const base = adminBase()
  if (!suffix) return base
  return base + (suffix.startsWith('/') ? suffix : `/${suffix}`)
}

export function isAdminLocation(pathname) {
  const base = adminBase()
  if (!base || typeof pathname !== 'string') return false
  return pathname === base || pathname.startsWith(`${base}/`)
}
