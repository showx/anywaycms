import axios from 'axios'
import { currentSiteHost } from './siteHost'

export const http = axios.create()

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('cms_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  const host = currentSiteHost()
  if (host) config.headers['X-Site-Host'] = host
  return config
})

http.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const status = error.response?.status
    const message = error.response?.data?.message || '网络请求失败'
    if (status === 401 && window.location.pathname.startsWith('/admin') && window.location.pathname !== '/admin/login') {
      localStorage.removeItem('cms_token')
      const redirect = encodeURIComponent(window.location.pathname + window.location.search)
      window.location.assign(`/admin/login?redirect=${redirect}`)
    }
    const wrapped = new Error(message)
    wrapped.status = status
    wrapped.host = error.response?.data?.host || ''
    return Promise.reject(wrapped)
  },
)
