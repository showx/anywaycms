<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { http } from '../../api'
import { adminPath } from '../../adminPath'
import { useContextStore } from '../../stores/context'

const ctx = useContextStore()
const rows = ref([])
const categories = ref([])
const articles = ref([])
const pages = ref([])
const error = ref('')
const saved = ref('')
const editing = ref(0)
const form = reactive(blank())

const linkOptions = computed(() => {
  if (form.link_type === 'category') return categories.value.map((item) => ({ id: item.id, label: item.name }))
  if (form.link_type === 'article') return articles.value.map((item) => ({ id: item.id, label: item.title }))
  if (form.link_type === 'page') return pages.value.map((item) => ({ id: item.id, label: item.title }))
  return []
})

function blank() {
  return { name: '', link_type: 'home', link_id: 0, url: '', sort: 0 }
}

function typeLabel(type) {
  return { home: '首页', category: '栏目', article: '文章', page: '单页', url: '外链' }[type] || type
}

function targetLabel(row) {
  if (row.link_type === 'home') return '首页'
  if (row.link_type === 'url') return row.url
  const list = row.link_type === 'category' ? categories.value : row.link_type === 'article' ? articles.value : pages.value
  const hit = list.find((item) => item.id === row.link_id)
  return hit ? (hit.name || hit.title) : '已失效'
}

function edit(row) {
  editing.value = row.id
  Object.assign(form, {
    name: row.name,
    link_type: row.link_type,
    link_id: row.link_id,
    url: row.url,
    sort: row.sort,
  })
}

function reset() {
  editing.value = 0
  Object.assign(form, blank())
}

async function load() {
  rows.value = []
  if (!ctx.siteId) return
  error.value = ''
  try {
    const [menuRes, catRes, articleRes, pageRes] = await Promise.all([
      http.get(`/api/sites/${ctx.siteId}/menus`),
      http.get(`/api/sites/${ctx.siteId}/categories`),
      http.get(`/api/sites/${ctx.siteId}/contents`, { params: { type: 'article', page_size: 200 } }),
      http.get(`/api/sites/${ctx.siteId}/contents`, { params: { type: 'page', page_size: 200 } }),
    ])
    rows.value = menuRes.data || []
    categories.value = catRes.data || []
    articles.value = articleRes.data.list || []
    pages.value = pageRes.data.list || []
  } catch (err) {
    error.value = err.message
  }
}

async function save() {
  error.value = ''
  saved.value = ''
  const payload = {
    name: form.name,
    link_type: form.link_type,
    link_id: Number(form.link_id || 0),
    url: form.url,
    sort: Number(form.sort || 0),
  }
  try {
    if (editing.value) await http.put(`/api/menus/${editing.value}`, payload)
    else await http.post(`/api/sites/${ctx.siteId}/menus`, payload)
    reset()
    saved.value = '已保存'
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(row) {
  if (!window.confirm(`删除菜单「${row.name}」？`)) return
  try {
    await http.delete(`/api/menus/${row.id}`)
    if (editing.value === row.id) reset()
    await load()
  } catch (err) {
    error.value = err.message
  }
}

watch(() => ctx.siteId, () => {
  reset()
  load()
}, { immediate: true })
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>菜单</h1>
        <p class="muted">前台导航来自这里。菜单项只能指向当前站点自己的栏目和内容。</p>
      </div>
    </div>
    <p v-if="!ctx.siteId" class="card empty">还没有站点。<router-link :to="adminPath('/sites')">去创建</router-link></p>
    <div v-else class="split">
      <form class="card" @submit.prevent="save">
        <h2>{{ editing ? '编辑菜单' : '新建菜单' }}</h2>
        <label class="field"><span>名称</span><input v-model="form.name" required /></label>
        <label class="field">
          <span>指向</span>
          <select v-model="form.link_type">
            <option value="home">首页</option>
            <option value="category">栏目</option>
            <option value="page">单页</option>
            <option value="article">文章</option>
            <option value="url">外链</option>
          </select>
        </label>
        <label v-if="linkOptions.length || ['category', 'article', 'page'].includes(form.link_type)" class="field">
          <span>目标</span>
          <select v-model.number="form.link_id">
            <option :value="0">请选择</option>
            <option v-for="item in linkOptions" :key="item.id" :value="item.id">{{ item.label }}</option>
          </select>
        </label>
        <label v-if="form.link_type === 'url'" class="field">
          <span>地址</span>
          <input v-model="form.url" placeholder="/about 或 https://example.com" />
        </label>
        <label class="field"><span>排序</span><input v-model.number="form.sort" type="number" /></label>
        <p v-if="error" class="msg error">{{ error }}</p>
        <p v-if="saved" class="msg ok">{{ saved }}</p>
        <div class="row-actions">
          <button class="btn primary" type="submit">保存</button>
          <button v-if="editing" class="btn" type="button" @click="reset">取消编辑</button>
        </div>
      </form>
      <div class="card">
        <table class="table" v-if="rows.length">
          <thead>
            <tr><th>名称</th><th>类型</th><th>指向</th><th>排序</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="row in rows" :key="row.id">
              <td>{{ row.name }}</td>
              <td>{{ typeLabel(row.link_type) }}</td>
              <td>{{ targetLabel(row) }}</td>
              <td>{{ row.sort }}</td>
              <td class="row-actions">
                <button class="btn small" type="button" @click="edit(row)">编辑</button>
                <button class="btn small danger" type="button" @click="remove(row)">删除</button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty">还没有菜单，前台会只显示一个首页。</p>
      </div>
    </div>
  </div>
</template>
