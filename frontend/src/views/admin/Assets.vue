<script setup>
import { ref, watch } from 'vue'
import { http } from '../../api'
import { useContextStore } from '../../stores/context'

const ctx = useContextStore()
const rows = ref([])
const error = ref('')
const saved = ref('')
const uploading = ref(false)
const fileEl = ref(null)

function sizeText(size) {
  const value = Number(size) || 0
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

function isImage(row) {
  return String(row.mime || '').startsWith('image/')
}

async function load() {
  rows.value = []
  error.value = ''
  if (!ctx.siteId) return
  const res = await http.get(`/api/sites/${ctx.siteId}/assets`)
  rows.value = res.data || []
}

async function onFile(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file || !ctx.siteId) return
  error.value = ''
  saved.value = ''
  uploading.value = true
  try {
    const body = new FormData()
    body.append('file', file)
    await http.post(`/api/sites/${ctx.siteId}/assets`, body)
    saved.value = '已上传'
    await load()
  } catch (err) {
    error.value = err.message
  } finally {
    uploading.value = false
  }
}

async function copy(row) {
  const full = new URL(row.url, window.location.origin).href
  try {
    await navigator.clipboard.writeText(full)
    saved.value = `已复制 ${full}`
  } catch {
    saved.value = full
  }
}

async function remove(row) {
  if (!window.confirm(`删除附件「${row.name}」？已经插入文章的图片会无法显示。`)) return
  error.value = ''
  try {
    await http.delete(`/api/assets/${row.id}`)
    await load()
  } catch (err) {
    error.value = err.message
  }
}

watch(() => ctx.siteId, () => {
  load().catch((err) => {
    error.value = err.message
  })
}, { immediate: true })
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>附件</h1>
        <p class="muted">图片和 PDF 只属于当前站点。上传后可插入文章，或复制地址放到封面里。</p>
      </div>
      <button class="btn primary" type="button" :disabled="!ctx.siteId || uploading" @click="fileEl?.click()">上传文件</button>
      <input ref="fileEl" class="hidden-file" type="file" accept="image/jpeg,image/png,image/gif,image/webp,application/pdf" @change="onFile" />
    </div>
    <p v-if="!ctx.siteId" class="card empty">先选择或创建一个站点。</p>
    <template v-else>
      <p v-if="error" class="msg error">{{ error }}</p>
      <p v-if="saved" class="msg ok">{{ saved }}</p>
      <div v-if="rows.length" class="asset-grid">
        <article v-for="row in rows" :key="row.id" class="asset-card">
          <img v-if="isImage(row)" :src="row.url" :alt="row.name" />
          <div v-else class="asset-file">PDF</div>
          <div class="meta">
            <strong>{{ row.name }}</strong>
            <small class="muted">{{ sizeText(row.size) }}</small>
            <div class="row-actions">
              <button class="btn small" type="button" @click="copy(row)">复制地址</button>
              <button class="btn small danger" type="button" @click="remove(row)">删除</button>
            </div>
          </div>
        </article>
      </div>
      <p v-else class="card empty">这个站点还没有附件。支持 JPG、PNG、GIF、WebP 和 PDF，单个不超过 8MB。</p>
    </template>
  </div>
</template>
