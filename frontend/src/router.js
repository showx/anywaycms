import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { setMeta } from './meta'
import PublicLayout from './layouts/PublicLayout.vue'
import AdminLayout from './layouts/AdminLayout.vue'
import Home from './views/public/Home.vue'
import Channel from './views/public/Channel.vue'
import Detail from './views/public/Detail.vue'
import Login from './views/admin/Login.vue'
import Dashboard from './views/admin/Dashboard.vue'
import Sites from './views/admin/Sites.vue'
import Categories from './views/admin/Categories.vue'
import Contents from './views/admin/Contents.vue'
import Assets from './views/admin/Assets.vue'
import HtmlSite from './views/admin/HtmlSite.vue'
import Menus from './views/admin/Menus.vue'
import Stats from './views/admin/Stats.vue'
import SEO from './views/admin/SEO.vue'
import Settings from './views/admin/Settings.vue'
import Password from './views/admin/Password.vue'
import { adminBase, adminPath, isAdminLocation } from './adminPath'

const ContentEdit = () => import('./views/admin/ContentEdit.vue')
const base = adminBase()
const routes = [
  {
    path: '/',
    component: PublicLayout,
    children: [
      { path: '', name: 'home', component: Home },
      { path: 'category/:slug', name: 'category', component: Channel },
      { path: 'article/:slug', name: 'article', component: Detail, meta: { kind: 'article' } },
      { path: 'page/:slug', name: 'page', component: Detail, meta: { kind: 'page' } },
    ],
  },
]
if (base) {
  routes.push(
    { path: `${base}/login`, name: 'login', component: Login },
    {
      path: base,
      component: AdminLayout,
      meta: { admin: true },
      children: [
        { path: '', name: 'dashboard', component: Dashboard },
        { path: 'sites', name: 'sites', component: Sites },
        { path: 'html', name: 'html-site', component: HtmlSite },
        { path: 'categories', name: 'categories', component: Categories },
        { path: 'contents', name: 'contents', component: Contents },
        { path: 'contents/new', name: 'content-new', component: ContentEdit },
        { path: 'contents/:id', name: 'content-edit', component: ContentEdit },
        { path: 'assets', name: 'assets', component: Assets },
        { path: 'menus', name: 'menus', component: Menus },
        { path: 'stats', name: 'stats', component: Stats },
        { path: 'seo', name: 'seo', component: SEO },
        { path: 'settings', name: 'settings', component: Settings },
        { path: 'password', name: 'password', component: Password },
      ],
    },
  )
}

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior() {
    return { top: 0 }
  },
  routes,
})

router.afterEach((to) => {
  if (isAdminLocation(to.path)) setMeta('robots', 'noindex,nofollow')
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.admin && !auth.token) {
    return { path: adminPath('/login'), query: { redirect: to.fullPath } }
  }
  if (to.path === adminPath('/login') && auth.token) {
    return { path: adminPath('') }
  }
  return true
})

export default router
