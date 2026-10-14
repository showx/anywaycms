<script setup>
import { computed, ref, watch } from 'vue'
import { http } from '../../api'
import { useContextStore } from '../../stores/context'

const ctx = useContextStore()
const files = ref([])
const error = ref('')
const saved = ref('')
const uploading = ref(false)
const editing = ref('')
const draft = ref('')
const newPath = ref('')
const zipEl = ref(null)
const folderEl = ref(null)

const htmlMode = computed(() => ctx.current?.mode === 'html')
const primaryHost = computed(() => (ctx.current?.domains || []).find((item) => item.is_primary)?.host || '')
const hasIndex = computed(() => files.value.some((item) => item.path === 'index.html' || item.path === 'index.htm'))
const previewHref = computed(() => {
  if (!primaryHost.value) return ''
  const query = `/?host=${encodeURIComponent(primaryHost.value)}`
  if (import.meta.env.DEV) return `http://127.0.0.1:8080${query}`
  return query
})

function sizeText(size) {
  const value = Number(size) || 0
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

async function load() {
  files.value = []
  editing.value = ''
  draft.value = ''
  error.value = ''
  if (!ctx.siteId) return
  const res = await http.get(`/api/sites/${ctx.siteId}/html`)
  files.value = res.data.files || []
}

async function setHTMLMode() {
  const site = ctx.current
  if (!site) return
  error.value = ''
  saved.value = ''
  try {
    await http.put(`/api/sites/${site.id}`, {
      name: site.name,
      title: site.title,
      subtitle: site.subtitle,
      description: site.description,
      keywords: site.keywords,
      phone: site.phone,
      address: site.address,
      icp: site.icp,
      theme: site.theme,
      status: site.status,
      mode: 'html',
    })
    await ctx.loadSites(true)
    saved.value = '已改为整站 HTML'
  } catch (err) {
    error.value = err.message
  }
}

async function onZip(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file || !ctx.siteId) return
  if (files.value.length && !window.confirm('上传压缩包会替换当前整站文件，继续吗？')) return
  await send(file, '', true)
}

async function onFolder(event) {
  const picked = [...(event.target.files || [])]
  event.target.value = ''
  if (!picked.length || !ctx.siteId) return
  const body = new FormData()
  let count = 0
  for (const file of picked) {
    const rel = folderRel(file)
    if (!rel) continue
    body.append('file', file, file.name)
    body.append('path', rel)
    count += 1
  }
  if (!count) {
    error.value = '文件夹里没有可上传的网页文件'
    return
  }
  error.value = ''
  saved.value = ''
  uploading.value = true
  try {
    const res = await http.post(`/api/sites/${ctx.siteId}/html`, body)
    saved.value = res.data?.has_index ? '文件夹已上传' : '已上传，但还没有 index.html，打开域名时看不到首页'
    await load()
  } catch (err) {
    error.value = err.message
  } finally {
    uploading.value = false
  }
}

function folderRel(file) {
  const raw = String(file.webkitRelativePath || file.name).replaceAll('\\', '/')
  const parts = raw.split('/').filter(Boolean)
  if (parts.some((part) => part.startsWith('.'))) return ''
  if (file.webkitRelativePath && parts.length > 1) return parts.slice(1).join('/')
  return parts.join('/')
}

async function send(file, rel, replaced) {
  error.value = ''
  saved.value = ''
  uploading.value = true
  try {
    const body = new FormData()
    body.append('file', file)
    if (rel) body.append('path', rel)
    const res = await http.post(`/api/sites/${ctx.siteId}/html`, body)
    if (replaced && res.data?.has_index === false) {
      saved.value = '压缩包已上传，但里面没有 index.html'
    } else {
      saved.value = replaced ? '整站已替换' : '已上传'
    }
    await load()
  } catch (err) {
    error.value = err.message
  } finally {
    uploading.value = false
  }
}

async function openFile(path) {
  error.value = ''
  try {
    const res = await http.get(`/api/sites/${ctx.siteId}/html/file`, { params: { path } })
    editing.value = res.data.path
    draft.value = res.data.content || ''
  } catch (err) {
    error.value = err.message
  }
}

function startNew() {
  const path = newPath.value.trim().replaceAll('\\', '/')
  if (!path) return
  const existing = files.value.find((item) => item.path === path)
  if (existing) {
    openFile(path)
    return
  }
  editing.value = path
  draft.value = '<!DOCTYPE html>\n<html lang="zh-CN">\n<head>\n  <meta charset="utf-8">\n  <title></title>\n</head>\n<body>\n\n</body>\n</html>\n'
}

async function saveFile() {
  if (!editing.value) return
  error.value = ''
  saved.value = ''
  try {
    await http.put(`/api/sites/${ctx.siteId}/html/file`, { path: editing.value, content: draft.value })
    saved.value = '已保存'
    const current = editing.value
    await load()
    await openFile(current)
  } catch (err) {
    error.value = err.message
  }
}

async function removeFile(path) {
  if (!window.confirm(`删除 ${path}？`)) return
  error.value = ''
  try {
    await http.delete(`/api/sites/${ctx.siteId}/html/file`, { params: { path } })
    if (editing.value === path) {
      editing.value = ''
      draft.value = ''
    }
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function clearAll() {
  if (!window.confirm('清空这个站点已上传的全部网页文件？')) return
  error.value = ''
  try {
    await http.delete(`/api/sites/${ctx.siteId}/html`)
    editing.value = ''
    draft.value = ''
    saved.value = '已清空'
    await load()
  } catch (err) {
    error.value = err.message
  }
}

watch(() => ctx.siteId, () => {
  saved.value = ''
  load().catch((err) => {
    error.value = err.message
  })
}, { immediate: true })
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>整站 HTML</h1>
        <p class="muted">把现成网站打包上传。绑定域名后直接打开这些文件，后台可以继续改 HTML。</p>
      </div>
      <div class="row-actions">
        <a v-if="previewHref && htmlMode" class="btn" :href="previewHref" target="_blank" rel="noopener">访问</a>
        <button class="btn primary" type="button" :disabled="!ctx.siteId || uploading" @click="zipEl?.click()">上传压缩包</button>
        <button class="btn" type="button" :disabled="!ctx.siteId || uploading" @click="folderEl?.click()">上传文件夹</button>
        <input ref="zipEl" class="hidden-file" type="file" accept=".zip,application/zip" @change="onZip" />
        <input ref="folderEl" class="hidden-file" type="file" webkitdirectory multiple @change="onFolder" />
      </div>
    </div>
    <p v-if="!ctx.siteId" class="card empty">先选择或创建一个站点。</p>
    <template v-else>
      <div v-if="!htmlMode" class="callout">
        当前站点还是栏目内容。改成整站 HTML 后，{{ primaryHost || '绑定域名' }} 会打开上传的页面，不再使用栏目和文章。
        <button class="btn small" type="button" @click="setHTMLMode">改为整站 HTML</button>
      </div>
      <div v-else class="callout">
        打开 {{ primaryHost || '绑定域名' }} 就是 <code>index.html</code>。其他页面按原来的路径访问，例如 <code>/about.html</code>、<code>/css/style.css</code>。后台入口不会被这些文件覆盖。
      </div>
      <p v-if="error" class="msg error">{{ error }}</p>
      <p v-if="saved" class="msg ok">{{ saved }}</p>
      <p v-if="files.length && !hasIndex" class="msg error">还没有 index.html，访客打开域名时看不到首页。</p>
      <div class="html-grid">
        <section class="card">
          <div class="section-head">
            <h2>文件</h2>
            <button v-if="files.length" class="btn small danger" type="button" @click="clearAll">清空</button>
          </div>
          <p v-if="!files.length" class="empty">还没有文件。上传 zip，或选择网站文件夹。压缩包里保留原来的目录。</p>
          <table v-else class="table">
            <thead>
              <tr><th>路径</th><th>大小</th><th></th></tr>
            </thead>
            <tbody>
              <tr v-for="item in files" :key="item.path">
                <td>{{ item.path }}</td>
                <td>{{ sizeText(item.size) }}</td>
                <td class="row-actions">
                  <button v-if="item.editable" class="btn small" type="button" @click="openFile(item.path)">编辑</button>
                  <button class="btn small danger" type="button" @click="removeFile(item.path)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
        </section>
        <section class="card">
          <h2>{{ editing || '编辑页面' }}</h2>
          <div class="domain-add" style="margin-top: 0; margin-bottom: 12px">
            <input v-model="newPath" placeholder="about.html" @keyup.enter="startNew" />
            <button class="btn" type="button" @click="startNew">新建</button>
          </div>
          <p v-if="!editing" class="empty">从左侧选择 HTML、CSS 或 JS，或新建一个页面。</p>
          <template v-else>
            <textarea class="code-editor" v-model="draft" spellcheck="false"></textarea>
            <div class="row-actions" style="margin-top: 12px">
              <button class="btn primary" type="button" @click="saveFile">保存</button>
            </div>
          </template>
        </section>
      </div>
    </template>
  </div>
</template>
