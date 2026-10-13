<script setup>
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { http } from '../api'
import { usePublicStore } from '../stores/public'
import { parseTheme } from '../theme'
import { publicPath, isPreviewing, exitPreview } from '../siteHost'
import { isAdminLocation } from '../adminPath'
import { applyHead } from '../meta'
import Missing from '../views/public/Missing.vue'

const route = useRoute()
const store = usePublicStore()
const site = computed(() => store.site || {})
const theme = computed(() => parseTheme(site.value.theme))
const previewing = computed(() => isPreviewing())
const browserHost = computed(() => window.location.hostname)
const mark = computed(() => (site.value.title || '站').slice(0, 1))
const domainText = computed(() => (site.value.domains || []).map((item) => item.host).join('、'))

function external(url) {
  return /^https?:\/\//.test(url || '')
}

let seoSeq = 0

function trackVisit() {
  if (isAdminLocation(route.path)) return
  http.post('/api/public/visit', { path: route.path || '/', referer: document.referrer || '' }).catch(() => {})
}

async function applyPageSEO() {
  const seq = ++seoSeq
  const path = route.path || '/'
  const page = route.query.page
  try {
    const res = await http.get('/api/public/seo', { params: { path, page: page || undefined } })
    if (seq !== seoSeq) return
    applyHead(res.data)
  } catch {
    // 域名未绑定或接口暂时失败时，保留服务器已经写好的标题。
  }
}

watch(() => route.fullPath, () => {
  store.load()
  trackVisit()
  applyPageSEO()
}, { immediate: true })
</script>

<template>
  <div class="public" :class="'layout-' + theme.layout" :style="{ '--brand': theme.primary }">
    <div v-if="store.loading" class="stage">正在打开站点…</div>
    <div v-else-if="store.error" class="wrap">
      <Missing />
    </div>
    <template v-else>
      <div v-if="previewing" class="preview-bar">
        <span>正在预览 {{ store.host }} 的「{{ site.title }}」。地址栏主机名是 {{ browserHost }}。</span>
        <button class="btn small" type="button" @click="exitPreview">回到当前主机名</button>
      </div>
      <div v-if="site.phone || site.address" class="topstrip">
        <div class="wrap strip-inner">
          <span>{{ site.phone }}</span>
          <span>{{ site.address }}</span>
        </div>
      </div>
      <header class="mast">
        <div class="wrap mast-inner">
          <router-link class="logo" :to="publicPath('/')">
            <span class="brand-mark">{{ mark }}</span>
            <span>
              <strong>{{ site.title }}</strong>
              <small v-if="site.subtitle">{{ site.subtitle }}</small>
            </span>
          </router-link>
          <nav class="nav">
            <router-link v-if="!store.menus.length" :to="publicPath('/')" exact-active-class="on">首页</router-link>
            <template v-for="item in store.menus" :key="item.id">
              <a v-if="external(item.url)" :href="item.url" target="_blank" rel="noopener">{{ item.name }}</a>
              <router-link v-else :to="publicPath(item.url)" exact-active-class="on">{{ item.name }}</router-link>
            </template>
          </nav>
        </div>
      </header>
      <main class="wrap">
        <router-view />
      </main>
      <footer class="footer">
        <div class="wrap">
          <strong>{{ site.title }}</strong>
          <p v-if="site.address">{{ site.address }}<span v-if="site.phone"> · {{ site.phone }}</span></p>
          <p>当前访问 {{ store.host }}，这个域名属于「{{ site.title }}」。</p>
          <p v-if="domainText">已绑定域名：{{ domainText }}</p>
          <p v-if="site.icp">{{ site.icp }}</p>
        </div>
      </footer>
    </template>
  </div>
</template>
