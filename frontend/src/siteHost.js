export function capturePreviewHost() {
  const params = new URLSearchParams(window.location.search)
  if (!params.has('host')) return
  const host = (params.get('host') || '').trim().toLowerCase()
  if (host) sessionStorage.setItem('cms_preview_host', host)
  else sessionStorage.removeItem('cms_preview_host')
}

export function currentSiteHost() {
  const query = new URLSearchParams(window.location.search).get('host')
  if (query && query.trim()) return query.trim().toLowerCase()
  const saved = sessionStorage.getItem('cms_preview_host')
  if (saved && saved !== window.location.hostname) return saved
  return window.location.hostname
}

export function isPreviewing() {
  return currentSiteHost() !== window.location.hostname
}

export function contentPath(kind, slug, mode) {
  const path = `/${kind}/${slug}`
  return mode === 'html' ? `${path}.html` : path
}

export function routeSlug(value) {
  return String(value || '').replace(/\.html$/i, '')
}

export function publicPath(path) {
  if (!path) return '/'
  if (/^https?:\/\//.test(path)) return path
  if (!isPreviewing()) return path
  const hashAt = path.indexOf('#')
  const hash = hashAt >= 0 ? path.slice(hashAt) : ''
  const withoutHash = hashAt >= 0 ? path.slice(0, hashAt) : path
  const [pathname, search = ''] = withoutHash.split('?')
  const params = new URLSearchParams(search)
  params.set('host', currentSiteHost())
  const query = params.toString()
  return `${pathname}${query ? `?${query}` : ''}${hash}`
}

export function previewQuery(extra = {}) {
  const query = { ...extra }
  if (isPreviewing()) query.host = currentSiteHost()
  return query
}

export function exitPreview() {
  sessionStorage.removeItem('cms_preview_host')
  window.location.href = '/'
}
