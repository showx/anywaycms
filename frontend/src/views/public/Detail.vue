<script setup>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { http } from '../../api'
import { usePublicStore } from '../../stores/public'
import { publicPath } from '../../siteHost'
import { setMeta, setTitle } from '../../meta'
import { formatTime } from '../../format'

const route = useRoute()
const store = usePublicStore()
const item = ref(null)
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  item.value = null
  const kind = route.meta.kind === 'page' ? 'pages' : 'articles'
  try {
    const res = await http.get(`/api/public/${kind}/${route.params.slug}`)
    item.value = res.data
    const siteTitle = store.site?.title || ''
    setTitle([item.value.seo_title || item.value.title, siteTitle].filter(Boolean).join(' - '))
    setMeta('description', item.value.seo_description || item.value.summary || '')
    setMeta('keywords', item.value.seo_keywords || store.site?.keywords || '')
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

watch(() => [route.params.slug, route.meta.kind, store.host], load, { immediate: true })
</script>

<template>
  <article class="article">
    <p v-if="loading" class="muted">正在读取正文…</p>
    <p v-else-if="error" class="msg error">{{ error }}</p>
    <template v-else-if="item">
      <nav class="crumb">
        <router-link :to="publicPath('/')">首页</router-link>
        <template v-if="item.category_slug">
          <span>/</span>
          <router-link :to="publicPath('/category/' + item.category_slug)">{{ item.category_name }}</router-link>
        </template>
      </nav>
      <h1>{{ item.title }}</h1>
      <p class="muted" v-if="item.published_at">{{ formatTime(item.published_at) }}</p>
      <p class="lede" v-if="item.summary">{{ item.summary }}</p>
      <div class="prose" v-html="item.body"></div>
    </template>
  </article>
</template>
