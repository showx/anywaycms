<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { http } from '../../api'
import { adminPath } from '../../adminPath'
import { useContextStore } from '../../stores/context'
import RichText from '../../components/RichText.vue'

const route = useRoute()
const router = useRouter()
const ctx = useContextStore()
const categories = ref([])
const error = ref('')
const saved = ref('')
const loading = ref(false)
const editorRef = ref(null)
const fileEl = ref(null)
const coverEl = ref(null)
const assets = ref([])
const form = reactive(blank('article'))

const isNew = computed(() => route.name === 'content-new')
const snippetTitle = computed(() => {
  const page = (form.seo_title || form.title || '未命名').trim()
  const site = ctx.current?.title || ''
  if (!site || page === site) return page
  return `${page} - ${site}`
})
const snippetDesc = computed(() => (form.seo_description || form.summary || '').trim() || '填写摘要或 SEO 描述后，这里会显示搜索结果里的说明。')
const snippetURL = computed(() => {
  const domains = ctx.current?.domains || []
  const primary = domains.find((item) => item.is_primary) || domains[0]
  const host = primary?.host || '主域名'
  const slug = form.slug || '…'
  const prefix = form.type === 'page' ? '/page/' : '/article/'
  const ext = ctx.current?.url_mode === 'html' ? '.html' : ''
  return `${host}${prefix}${slug}${ext}`
})

function blank(type) {
  return {
    type,
    title: '',
    slug: '',
    category_id: 0,
    summary: '',
    cover: '',
    body: '',
    status: 'published',
    seo_title: '',
    seo_keywords: '',
    seo_description: '',
    sort: 0,
  }
}

async function loadCategories() {
  if (!ctx.siteId) {
    categories.value = []
    return
  }
  const res = await http.get(`/api/sites/${ctx.siteId}/categories`)
  categories.value = res.data || []
}

async function loadAssets() {
  if (!ctx.siteId) {
    assets.value = []
    return
  }
  const res = await http.get(`/api/sites/${ctx.siteId}/assets`)
  assets.value = (res.data || []).filter((item) => String(item.mime || '').startsWith('image/')).slice(0, 12)
}

async function load() {
  error.value = ''
  loading.value = true
  try {
    await loadCategories()
    await loadAssets()
    if (isNew.value) {
      Object.assign(form, blank(route.query.type === 'page' ? 'page' : 'article'))
      return
    }
    const res = await http.get(`/api/contents/${route.params.id}`)
    Object.assign(form, blank(res.data.type), res.data)
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

function placeAsset(asset, asCover) {
  if (asCover) {
    form.cover = asset.url
    return
  }
  editorRef.value?.insertAsset(asset)
}

async function uploadPicked(event, asCover) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file || !ctx.siteId) return
  error.value = ''
  try {
    const body = new FormData()
    body.append('file', file)
    const res = await http.post(`/api/sites/${ctx.siteId}/assets`, body)
    placeAsset(res.data, asCover)
    await loadAssets()
  } catch (err) {
    error.value = err.message
  }
}

async function save() {
  error.value = ''
  saved.value = ''
  const payload = {
    type: form.type,
    title: form.title,
    slug: form.slug,
    category_id: form.type === 'page' ? 0 : Number(form.category_id || 0),
    summary: form.summary,
    cover: form.cover,
    body: editorRef.value?.html() ?? form.body,
    status: form.status,
    seo_title: form.seo_title,
    seo_keywords: form.seo_keywords,
    seo_description: form.seo_description,
    sort: Number(form.sort || 0),
  }
  try {
    if (isNew.value) {
      const res = await http.post(`/api/sites/${ctx.siteId}/contents`, payload)
      saved.value = '已创建'
      await router.replace(adminPath(`/contents/${res.data.id}`))
      return
    }
    await http.put(`/api/contents/${route.params.id}`, payload)
    saved.value = '已保存'
  } catch (err) {
    error.value = err.message
  }
}

watch(() => route.fullPath, () => load(), { immediate: true })
watch(() => ctx.siteId, (next, prev) => {
  if (prev && next !== prev) router.push(adminPath('/contents'))
})
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ isNew ? '新建内容' : '编辑内容' }}</h1>
        <p class="muted">正文直接排版。保存后只会出现在当前站点。</p>
      </div>
      <router-link class="btn" :to="adminPath('/contents')">返回列表</router-link>
    </div>
    <p v-if="!ctx.siteId" class="card empty">先选择或创建一个站点。</p>
    <form v-else class="card" @submit.prevent="save">
      <div class="form-grid">
        <label class="field">
          <span>类型</span>
          <select v-model="form.type">
            <option value="article">文章</option>
            <option value="page">单页</option>
          </select>
        </label>
        <label class="field">
          <span>状态</span>
          <select v-model="form.status">
            <option value="published">发布</option>
            <option value="draft">草稿</option>
          </select>
        </label>
        <label class="field wide"><span>标题</span><input v-model="form.title" required /></label>
        <label class="field"><span>别名</span><input v-model="form.slug" placeholder="可留空，保存后自动生成" /></label>
        <label v-if="form.type === 'article'" class="field">
          <span>栏目</span>
          <select v-model.number="form.category_id">
            <option :value="0">请选择</option>
            <option v-for="cat in categories" :key="cat.id" :value="cat.id">{{ cat.name }}</option>
          </select>
        </label>
        <label class="field wide"><span>摘要</span><textarea class="hosts" v-model="form.summary"></textarea></label>
        <label class="field wide">
          <span>封面地址</span>
          <input v-model="form.cover" placeholder="https:// 或站内路径，可留空" />
        </label>
        <div class="field wide">
          <button class="btn small" type="button" @click="coverEl?.click()">上传封面</button>
          <input ref="coverEl" class="hidden-file" type="file" accept="image/jpeg,image/png,image/gif,image/webp" @change="uploadPicked($event, true)" />
        </div>
      </div>
      <p v-if="form.type === 'article' && categories.length === 0" class="msg">这个站点还没有栏目。<router-link :to="adminPath('/categories')">先去创建栏目</router-link></p>
      <RichText ref="editorRef" v-model="form.body">
        <template #tools>
          <button class="btn small" type="button" @click="fileEl?.click()">插入图片</button>
          <input ref="fileEl" class="hidden-file" type="file" accept="image/jpeg,image/png,image/gif,image/webp,application/pdf" @change="uploadPicked($event, false)" />
        </template>
      </RichText>
      <div v-if="assets.length" class="pick-row">
        <button v-for="asset in assets" :key="asset.id" type="button" :title="asset.name" @click="placeAsset(asset, false)">
          <img :src="asset.url" :alt="asset.name" />
        </button>
      </div>
      <details class="seo">
        <summary>搜索引擎信息</summary>
        <div class="snippet">
          <small>{{ snippetURL }}</small>
          <strong>{{ snippetTitle }}</strong>
          <p>{{ snippetDesc }}</p>
        </div>
        <div class="form-grid">
          <label class="field"><span>SEO 标题</span><input v-model="form.seo_title" placeholder="留空则使用内容标题" /></label>
          <label class="field"><span>SEO 关键词</span><input v-model="form.seo_keywords" /></label>
          <label class="field wide"><span>SEO 描述</span><textarea class="hosts" v-model="form.seo_description" placeholder="留空则使用摘要"></textarea></label>
        </div>
      </details>
      <p v-if="error" class="msg error">{{ error }}</p>
      <p v-if="saved" class="msg ok">{{ saved }}</p>
      <button class="btn primary" type="submit" :disabled="loading">保存</button>
    </form>
  </div>
</template>
