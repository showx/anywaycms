export function setMeta(name, content) {
  let el = document.head.querySelector(`meta[name="${name}"]`)
  if (!content) {
    el?.remove()
    return
  }
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute('name', name)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

function setProperty(name, content) {
  let el = document.head.querySelector(`meta[property="${name}"]`)
  if (!content) {
    el?.remove()
    return
  }
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute('property', name)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

export function setTitle(title) {
  document.title = title || '站群 CMS'
}

export function applyHead(doc) {
  const data = doc || {}
  setTitle(data.title)
  setMeta('description', data.description || '')
  setMeta('keywords', data.keywords || '')
  setMeta('robots', data.robots || '')
  setProperty('og:title', data.title || '')
  setProperty('og:description', data.description || '')
  setProperty('og:url', data.canonical || '')
  setProperty('og:type', data.og_type || '')
  setProperty('og:site_name', data.site_name || '')
  setProperty('og:image', data.image || '')
  setMeta('twitter:card', data.title ? 'summary' : '')
  setMeta('twitter:title', data.title || '')
  setMeta('twitter:description', data.description || '')
  let link = document.head.querySelector('link[rel="canonical"]')
  if (data.canonical) {
    if (!link) {
      link = document.createElement('link')
      link.rel = 'canonical'
      document.head.appendChild(link)
    }
    link.href = data.canonical
  } else {
    link?.remove()
  }
  let script = document.getElementById('seo-jsonld')
  if (data.json_ld) {
    if (!script) {
      script = document.createElement('script')
      script.id = 'seo-jsonld'
      script.type = 'application/ld+json'
      document.head.appendChild(script)
    }
    script.textContent = data.json_ld
  } else {
    script?.remove()
  }
}
