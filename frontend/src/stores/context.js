import { defineStore } from 'pinia'
import { http } from '../api'

function readSiteId() {
  const value = Number(localStorage.getItem('cms_site_id') || 0)
  return Number.isFinite(value) ? value : 0
}

let inflight = null

export const useContextStore = defineStore('context', {
  state: () => ({
    sites: [],
    siteId: readSiteId(),
    ready: false,
  }),
  getters: {
    current(state) {
      return state.sites.find((site) => site.id === state.siteId) || null
    },
  },
  actions: {
    loadSites(force = false) {
      if (inflight && !force) return inflight
      inflight = this.fetchSites().finally(() => {
        inflight = null
      })
      return inflight
    },
    async fetchSites() {
      const res = await http.get('/api/sites')
      this.sites = res.data || []
      const exists = this.sites.some((site) => site.id === this.siteId)
      if (!exists) this.setSite(this.sites[0]?.id || 0)
      this.ready = true
    },
    setSite(id) {
      this.siteId = id
      localStorage.setItem('cms_site_id', String(id || 0))
    },
  },
})
