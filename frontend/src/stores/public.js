import { defineStore } from 'pinia'
import { http } from '../api'
import { currentSiteHost } from '../siteHost'

export const usePublicStore = defineStore('public', {
  state: () => ({
    site: null,
    menus: [],
    categories: [],
    host: '',
    boundHost: '',
    loading: true,
    loaded: false,
    error: '',
  }),
  actions: {
    async load(force = false) {
      const host = currentSiteHost()
      if (!force && this.loaded && !this.error && this.boundHost === host) {
        this.loading = false
        return
      }
      if (!this.site) this.loading = true
      try {
        const res = await http.get('/api/public/bootstrap')
        this.site = res.data.site
        this.menus = res.data.menus || []
        this.categories = res.data.categories || []
        this.host = res.data.host || host
        this.boundHost = this.host
        this.error = ''
        this.loaded = true
      } catch (error) {
        this.site = null
        this.menus = []
        this.categories = []
        this.error = error.message
        this.host = error.host || host
        this.boundHost = host
        this.loaded = false
      } finally {
        this.loading = false
      }
    },
  },
})
