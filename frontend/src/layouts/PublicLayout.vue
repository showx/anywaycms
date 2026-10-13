<script setup>
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { http } from '../api'
import { usePublicStore } from '../stores/public'
import { parseTheme } from '../theme'
import { publicPath, isPreviewing, exitPreview } from '../siteHost'
import { setMeta, setTitle } from '../meta'
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

function trackVisit() {
  if (route.path.startsWith('/admin')) return
  http.post('/api/public/visit', { path: route.path || '/', referer: document.referrer || '' }).catch(() => {})
}

watch(() => route.fullPath, () => {
  store.load()
  trackVisit()
}, { immediate: true })
watch(site, (value) => {
  if (!value?.title || route.name !== 'home') return
  setTitle(value.title)
  setMeta('description', value.description || '')
  setMeta('keywords', value.keywords || '')
})
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
