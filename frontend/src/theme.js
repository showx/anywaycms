export const themePresets = [
  { id: 'corporate-teal', label: '企业青绿', layout: 'corporate', primary: '#0f6e6b' },
  { id: 'corporate-navy', label: '企业海军蓝', layout: 'corporate', primary: '#1e3a5f' },
  { id: 'magazine-clay', label: '文旅赭石', layout: 'magazine', primary: '#9a3412' },
  { id: 'magazine-lake', label: '杂志湖蓝', layout: 'magazine', primary: '#0369a1' },
]

export function parseTheme(raw) {
  const fallback = { layout: 'corporate', primary: '#0f6e6b' }
  try {
    const data = JSON.parse(raw || '{}')
    return {
      layout: data.layout === 'magazine' ? 'magazine' : 'corporate',
      primary: /^#[0-9a-fA-F]{6}$/.test(data.primary || '') ? data.primary : fallback.primary,
    }
  } catch {
    return fallback
  }
}

export function themeJSON(layout, primary) {
  const safeLayout = layout === 'magazine' ? 'magazine' : 'corporate'
  const safePrimary = /^#[0-9a-fA-F]{6}$/.test(primary || '') ? primary : '#0f6e6b'
  return JSON.stringify({ layout: safeLayout, primary: safePrimary })
}
