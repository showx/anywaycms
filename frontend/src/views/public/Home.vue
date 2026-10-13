<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { http } from '../../api'
import { usePublicStore } from '../../stores/public'
import { publicPath } from '../../siteHost'
import { parseTheme } from '../../theme'
import { setMeta, setTitle } from '../../meta'
import { formatTime } from '../../format'

const store = usePublicStore()
const latest = ref([])
const blocks = ref([])
const pages = ref([])
const error = ref('')
const loading = ref(true)
const layout = computed(() => parseTheme(store.site?.theme).layout)
const shownLatest = computed(() => (layout.value === 'magazine' ? latest.value.slice(1) : latest.value))

async function load() {
  if (!store.site) return
  loading.value = true
  error.value = ''
  try {
    const res = await http.get('/api/public/home')
    latest.value = res.data.latest || []
    blocks.value = res.data.blocks || []
    pages.value = res.data.pages || []
    setTitle(store.site.title)
    setMeta('description', store.site.description || '')
    setMeta('keywords', store.site.keywords || '')
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => store.host, load)
</script>

<template>
  <div>
    <p v-if="loading" class="muted">正在读取页面…</p>
    <p v-else-if="error" class="msg error">{{ error }}</p>
    <template v-else>
      <section v-if="layout === 'corporate'" class="hero">
        <p class="eyebrow">{{ store.site.subtitle }}</p>
        <h1>{{ store.site.title }}</h1>
        <p class="lede">{{ store.site.description }}</p>
      </section>

      <section v-if="layout === 'magazine' && latest[0]" class="feature">
        <p class="eyebrow">{{ latest[0].category_name || '最新' }}</p>
        <h1>{{ latest[0].title }}</h1>
        <p class="lede">{{ latest[0].summary }}</p>
        <router-link :to="publicPath('/article/' + latest[0].slug)">阅读这篇</router-link>
      </section>

      <section class="section" v-if="shownLatest.length">
        <div class="section-head">
          <h2>{{ layout === 'magazine' ? '更多内容' : '最新文章' }}</h2>
        </div>
        <div class="card-grid">
          <router-link v-for="item in shownLatest" :key="item.id" class="story" :to="publicPath('/article/' + item.slug)">
            <p class="kicker">{{ item.category_name || '文章' }}</p>
            <h3>{{ item.title }}</h3>
            <p>{{ item.summary }}</p>
            <small class="muted">{{ formatTime(item.published_at) }}</small>
          </router-link>
        </div>
      </section>

      <section v-for="block in blocks" :key="block.category.id" class="section">
        <div class="section-head">
          <div>
            <h2>{{ block.category.name }}</h2>
            <p class="muted">{{ block.category.description }}</p>
          </div>
          <router-link :to="publicPath('/category/' + block.category.slug)">查看全部</router-link>
        </div>
        <div v-if="block.articles.length" class="card-grid">
          <router-link v-for="item in block.articles" :key="item.id" class="story" :to="publicPath('/article/' + item.slug)">
            <h3>{{ item.title }}</h3>
            <p>{{ item.summary }}</p>
          </router-link>
        </div>
        <p v-else class="muted">这个栏目还没有文章。</p>
      </section>

      <section v-if="pages.length" class="section">
        <h2>页面</h2>
        <div class="page-links">
          <router-link v-for="item in pages" :key="item.id" :to="publicPath('/page/' + item.slug)">{{ item.title }}</router-link>
        </div>
      </section>
    </template>
  </div>
</template>
