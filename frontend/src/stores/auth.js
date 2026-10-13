import { defineStore } from 'pinia'
import { http } from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('cms_token') || '',
    username: localStorage.getItem('cms_username') || '',
  }),
  actions: {
    async login(username, password) {
      const res = await http.post('/api/auth/login', { username, password })
      this.token = res.data.token
      this.username = res.data.username
      localStorage.setItem('cms_token', res.data.token)
      localStorage.setItem('cms_username', res.data.username)
    },
    logout() {
      this.token = ''
      this.username = ''
      localStorage.removeItem('cms_token')
      localStorage.removeItem('cms_username')
    },
  },
})
