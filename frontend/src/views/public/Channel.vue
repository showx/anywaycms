<script setup>
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { http } from '../../api'
import { usePublicStore } from '../../stores/public'
import { contentPath, publicPath, previewQuery, routeSlug } from '../../siteHost'
import { formatTime } from '../../format'

const route = useRoute()
const router = useRouter()
const store = usePublicStore()
const category = ref(null)
const list = ref([])
const total = ref(0)
const pageSize = ref(20)
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const page = Math.max(1, Number(route.query.page) || 1)
    const res = await http.get(`/api/public/categories/${routeSlug(route.params.slug)}`, {
      params: { page, page_size: 12 },
    })
    category.value = res.data.category
    list.value = res.data.list || []
    total.value = res.data.total || 0
    pageSize.value = res.data.page_size || 12
  } catch (err) {
    category.value = null
    list.value = []
    error.value = err.message
  } finally {
    loading.value = false
  }
}

function go(page) {
  router.push({
    name: 'category',
    params: { slug: route.params.slug },
    query: previewQuery(page > 1 ? { page } : {}),
  })
}

watch(() => [route.params.slug, route.query.page, store.host], load, { immediate: true })
</script>

<template>
  <section class="section">
    <p v-if="loading" class="muted">正在读取栏目…</p>
    <p v-else-if="error" class="msg error">{{ error }}</p>
    <template v-else-if="category">
      <header>
        <h1>{{ category.name }}</h1>
        <p class="lede">{{ category.description }}</p>
      </header>
      <div v-if="list.length" class="card-grid">
        <router-link v-for="item in list" :key="item.id" class="story" :to="publicPath(contentPath('article', item.slug, store.site?.url_mode))">
          <h3>{{ item.title }}</h3>
          <p>{{ item.summary }}</p>
          <small class="muted">{{ formatTime(item.published_at) }}</small>
        </router-link>
      </div>
      <p v-else class="muted">这个栏目还没有发布文章。</p>
      <div class="pager" v-if="total > pageSize">
        <button class="btn small" type="button" :disabled="Number(route.query.page || 1) <= 1" @click="go(Number(route.query.page || 1) - 1)">上一页</button>
        <span>共 {{ total }} 篇</span>
        <button class="btn small" type="button" :disabled="Number(route.query.page || 1) * pageSize >= total" @click="go(Number(route.query.page || 1) + 1)">下一页</button>
      </div>
    </template>
  </section>
</template>
