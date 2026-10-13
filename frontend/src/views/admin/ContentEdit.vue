<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { http } from '../../api'
import { useContextStore } from '../../stores/context'

const route = useRoute()
const router = useRouter()
const ctx = useContextStore()
const categories = ref([])
const error = ref('')
const saved = ref('')
const loading = ref(false)
const bodyEl = ref(null)
const form = reactive(blank('article'))

const isNew = computed(() => route.name === 'content-new')

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

async function load() {
  error.value = ''
  loading.value = true
  try {
    await loadCategories()
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

function insertLink() {
  insert('<a href="https://">', '</a>')
}

function insert(before, after) {
  const el = bodyEl.value
  const picked = '文字'
  if (!el) {
    form.body += before + picked + after
    return
  }
  const start = el.selectionStart ?? form.body.length
  const end = el.selectionEnd ?? form.body.length
  const selected = form.body.slice(start, end) || picked
  form.body = form.body.slice(0, start) + before + selected + after + form.body.slice(end)
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
    body: form.body,
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
      await router.replace(`/admin/contents/${res.data.id}`)
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
  if (prev && next !== prev) router.push('/admin/contents')
})
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>{{ isNew ? '新建内容' : '编辑内容' }}</h1>
        <p class="muted">正文使用 HTML。保存后只会出现在当前站点。</p>
      </div>
      <router-link class="btn" to="/admin/contents">返回列表</router-link>
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
        <label class="field wide"><span>封面地址</span><input v-model="form.cover" placeholder="https:// 或站内路径，可留空" /></label>
      </div>
      <p v-if="form.type === 'article' && categories.length === 0" class="msg">这个站点还没有栏目。<router-link to="/admin/categories">先去创建栏目</router-link></p>
      <div class="editor-tools">
        <button class="btn small" type="button" @click="insert('<h2>', '</h2>')">标题</button>
        <button class="btn small" type="button" @click="insert('<p>', '</p>')">段落</button>
        <button class="btn small" type="button" @click="insert('<ul>\n  <li>', '</li>\n</ul>')">列表</button>
        <button class="btn small" type="button" @click="insertLink">链接</button>
      </div>
      <div class="editor">
        <textarea ref="bodyEl" v-model="form.body" placeholder="在这里写 HTML 正文"></textarea>
        <div class="prose preview" v-html="form.body || '<p class=&quot;muted&quot;>正文预览</p>'"></div>
      </div>
      <details class="seo">
        <summary>搜索引擎信息</summary>
        <div class="form-grid">
          <label class="field"><span>SEO 标题</span><input v-model="form.seo_title" /></label>
          <label class="field"><span>SEO 关键词</span><input v-model="form.seo_keywords" /></label>
          <label class="field wide"><span>SEO 描述</span><textarea class="hosts" v-model="form.seo_description"></textarea></label>
        </div>
      </details>
      <p v-if="error" class="msg error">{{ error }}</p>
      <p v-if="saved" class="msg ok">{{ saved }}</p>
      <button class="btn primary" type="submit" :disabled="loading">保存</button>
    </form>
  </div>
</template>
