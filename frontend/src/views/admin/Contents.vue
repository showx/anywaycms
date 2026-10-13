<script setup>
import { ref, watch } from 'vue'
import { http } from '../../api'
import { adminPath } from '../../adminPath'
import { useContextStore } from '../../stores/context'
import { formatTime } from '../../format'

const ctx = useContextStore()
const type = ref('article')
const q = ref('')
const page = ref(1)
const total = ref(0)
const rows = ref([])
const error = ref('')
const pageSize = 20

async function load() {
  rows.value = []
  total.value = 0
  if (!ctx.siteId) return
  error.value = ''
  try {
    const res = await http.get(`/api/sites/${ctx.siteId}/contents`, {
      params: { type: type.value, page: page.value, page_size: pageSize, q: q.value },
    })
    rows.value = res.data.list || []
    total.value = res.data.total || 0
  } catch (err) {
    error.value = err.message
  }
}

function previewHref(row) {
  const site = ctx.current
  const domain = (site?.domains || []).find((item) => item.is_primary) || (site?.domains || [])[0]
  const path = `/${row.type === 'page' ? 'page' : 'article'}/${row.slug}`
  if (!domain) return path
  return `${path}?host=${encodeURIComponent(domain.host)}`
}

function search() {
  page.value = 1
  load()
}

function switchType(next) {
  type.value = next
  page.value = 1
  q.value = ''
  load()
}

async function remove(row) {
  if (!window.confirm(`删除「${row.title}」？`)) return
  error.value = ''
  try {
    await http.delete(`/api/contents/${row.id}`)
    await load()
  } catch (err) {
    error.value = err.message
  }
}

watch(() => ctx.siteId, () => {
  page.value = 1
  load()
}, { immediate: true })
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>内容</h1>
        <p class="muted">文章进栏目，单页适合关于、联系这种固定页面。只发布当前站点的内容。</p>
      </div>
      <router-link v-if="ctx.siteId" class="btn primary" :to="adminPath(`/contents/new?type=${type}`)">
        新建{{ type === 'article' ? '文章' : '单页' }}
      </router-link>
    </div>
    <p v-if="!ctx.siteId" class="card empty">还没有站点。<router-link :to="adminPath('/sites')">去创建</router-link></p>
    <div v-else class="card">
      <div class="tabs">
        <button class="btn" :class="{ on: type === 'article' }" type="button" @click="switchType('article')">文章</button>
        <button class="btn" :class="{ on: type === 'page' }" type="button" @click="switchType('page')">单页</button>
        <input v-model="q" placeholder="搜索标题，回车确认" @keyup.enter="search" />
      </div>
      <p v-if="error" class="msg error">{{ error }}</p>
      <table class="table" v-if="rows.length">
        <thead>
          <tr>
            <th>标题</th>
            <th v-if="type === 'article'">栏目</th>
            <th>状态</th>
            <th>更新</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id">
            <td>
              <strong>{{ row.title }}</strong>
              <div class="muted">/{{ row.type === 'page' ? 'page' : 'article' }}/{{ row.slug }}</div>
            </td>
            <td v-if="type === 'article'">{{ row.category_name || '—' }}</td>
            <td><span class="pill" :data-on="row.status === 'published'">{{ row.status === 'published' ? '已发布' : '草稿' }}</span></td>
            <td>{{ formatTime(row.updated_at) }}</td>
            <td class="row-actions">
              <router-link class="btn small" :to="adminPath(`/contents/${row.id}`)">编辑</router-link>
              <a class="btn small" :href="previewHref(row)" target="_blank" rel="noopener">查看</a>
              <button class="btn small danger" type="button" @click="remove(row)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="empty">没有内容。</p>
      <div class="pager" v-if="total > pageSize">
        <button class="btn small" type="button" :disabled="page <= 1" @click="page -= 1; load()">上一页</button>
        <span>{{ page }} / {{ Math.ceil(total / pageSize) }}</span>
        <button class="btn small" type="button" :disabled="page >= Math.ceil(total / pageSize)" @click="page += 1; load()">下一页</button>
      </div>
    </div>
  </div>
</template>
