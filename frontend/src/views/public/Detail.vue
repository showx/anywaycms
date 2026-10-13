<script setup>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { http } from '../../api'
import { usePublicStore } from '../../stores/public'
import { contentPath, publicPath, routeSlug } from '../../siteHost'
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
    const res = await http.get(`/api/public/${kind}/${routeSlug(route.params.slug)}`)
    item.value = res.data
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
          <router-link :to="publicPath(contentPath('category', item.category_slug, store.site?.url_mode))">{{ item.category_name }}</router-link>
        </template>
      </nav>
      <h1>{{ item.title }}</h1>
      <p class="muted" v-if="item.published_at">{{ formatTime(item.published_at) }}</p>
      <p class="lede" v-if="item.summary">{{ item.summary }}</p>
      <div class="prose" v-html="item.body"></div>
    </template>
  </article>
</template>
