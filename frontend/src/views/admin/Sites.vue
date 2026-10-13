<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { http } from '../../api'
import { adminPath } from '../../adminPath'
import { useContextStore } from '../../stores/context'
import { parseTheme, themeJSON, themePresets } from '../../theme'

const ctx = useContextStore()
const selected = ref(0)
const draft = ref(false)
const error = ref('')
const saved = ref('')
const newHost = ref('')
const form = reactive(blank())

const current = computed(() => ctx.sites.find((site) => site.id === selected.value) || null)
const enabled = computed({
  get: () => form.status === 1,
  set: (value) => {
    form.status = value ? 1 : 0
  },
})

function blank() {
  return {
    name: '',
    title: '',
    subtitle: '',
    description: '',
    keywords: '',
    phone: '',
    address: '',
    icp: '',
    status: 1,
    layout: 'corporate',
    primary: '#0f6e6b',
    hostsText: '',
  }
}

function fill(site) {
  const theme = parseTheme(site.theme)
  Object.assign(form, {
    name: site.name,
    title: site.title,
    subtitle: site.subtitle,
    description: site.description,
    keywords: site.keywords,
    phone: site.phone,
    address: site.address,
    icp: site.icp,
    status: site.status,
    layout: theme.layout,
    primary: theme.primary,
    hostsText: '',
  })
}

function select(site) {
  draft.value = false
  selected.value = site.id
  fill(site)
  error.value = ''
  saved.value = ''
}

function createNew() {
  draft.value = true
  selected.value = 0
  Object.assign(form, blank())
  error.value = ''
  saved.value = ''
}

function applyPreset(preset) {
  form.layout = preset.layout
  form.primary = preset.primary
}

function payload() {
  return {
    name: form.name,
    title: form.title,
    subtitle: form.subtitle,
    description: form.description,
    keywords: form.keywords,
    phone: form.phone,
    address: form.address,
    icp: form.icp,
    theme: themeJSON(form.layout, form.primary),
    status: form.status,
  }
}

async function save() {
  error.value = ''
  saved.value = ''
  try {
    if (draft.value) {
      const hosts = form.hostsText.split(/[\n,]+/).map((item) => item.trim()).filter(Boolean)
      const res = await http.post('/api/sites', { ...payload(), hosts })
      draft.value = false
      selected.value = res.data.id
      ctx.setSite(res.data.id)
    } else {
      await http.put(`/api/sites/${selected.value}`, payload())
    }
    await ctx.loadSites(true)
    const fresh = ctx.sites.find((site) => site.id === selected.value)
    if (fresh) fill(fresh)
    saved.value = '已保存'
  } catch (err) {
    error.value = err.message
  }
}

async function addDomain() {
  error.value = ''
  try {
    await http.post(`/api/sites/${selected.value}/domains`, { host: newHost.value, is_primary: false })
    newHost.value = ''
    await ctx.loadSites(true)
    saved.value = '域名已绑定'
  } catch (err) {
    error.value = err.message
  }
}

async function makePrimary(id) {
  error.value = ''
  try {
    await http.put(`/api/domains/${id}/primary`)
    await ctx.loadSites(true)
  } catch (err) {
    error.value = err.message
  }
}

async function removeDomain(domain) {
  if (!window.confirm(`解除域名 ${domain.host}？`)) return
  error.value = ''
  try {
    await http.delete(`/api/domains/${domain.id}`)
    await ctx.loadSites(true)
  } catch (err) {
    error.value = err.message
  }
}

async function removeSite() {
  if (!current.value) return
  if (!window.confirm(`删除「${current.value.title}」？该站点的域名、栏目、内容和菜单都会一起删除。`)) return
  error.value = ''
  try {
    await http.delete(`/api/sites/${current.value.id}`)
    draft.value = false
    selected.value = 0
    await ctx.loadSites(true)
    if (ctx.sites[0]) select(ctx.sites[0])
  } catch (err) {
    error.value = err.message
  }
}

onMounted(async () => {
  await ctx.loadSites()
  const preferred = ctx.sites.find((site) => site.id === ctx.siteId) || ctx.sites[0]
  if (preferred) select(preferred)
})
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>站点与域名</h1>
        <p class="muted">每个站点是一个独立网站。把不同域名绑到不同站点，访客就会看到不同页面。</p>
      </div>
      <button class="btn primary" type="button" @click="createNew">新建站点</button>
    </div>
    <div class="callout">
      请求进来时先看域名。命中哪条绑定，就只渲染那个站点的菜单、栏目和文章。同一个网站的主域名和 www 可以绑在一起；要做另一个网站，就新建站点再绑新域名。
    </div>
    <div class="split">
      <aside class="site-nav">
        <button v-if="draft" class="site-item on" type="button">
          新站点
          <small>尚未保存</small>
        </button>
        <button
          v-for="site in ctx.sites"
          :key="site.id"
          class="site-item"
          :class="{ on: site.id === selected && !draft }"
          type="button"
          @click="select(site)"
        >
          {{ site.title }}
          <small>{{ (site.domains || []).find((item) => item.is_primary)?.host || '未绑定域名' }}</small>
        </button>
      </aside>
      <section class="card" v-if="draft || current">
        <div class="form-grid">
          <label class="field"><span>站点名称</span><input v-model="form.name" placeholder="内部识别用" /></label>
          <label class="field"><span>网站标题</span><input v-model="form.title" placeholder="访客看到的站名" /></label>
          <label class="field"><span>副标题</span><input v-model="form.subtitle" /></label>
          <label class="field"><span>电话</span><input v-model="form.phone" /></label>
          <label class="field wide"><span>简介</span><textarea class="hosts" v-model="form.description"></textarea></label>
          <label class="field"><span>关键词</span><input v-model="form.keywords" /></label>
          <p class="muted wide">搜索结果标题、是否收录和站点地图在 <router-link :to="adminPath('/seo')">SEO</router-link> 里设置。</p>
          <label class="field"><span>备案号</span><input v-model="form.icp" /></label>
          <label class="field wide"><span>地址</span><input v-model="form.address" /></label>
          <div class="field wide">
            <span>外观</span>
            <div class="preset-row">
              <button v-for="preset in themePresets" :key="preset.id" class="btn small" type="button" @click="applyPreset(preset)">
                {{ preset.label }}
              </button>
              <select v-model="form.layout">
                <option value="corporate">企业布局</option>
                <option value="magazine">杂志布局</option>
              </select>
              <input class="color-input" type="color" v-model="form.primary" />
            </div>
          </div>
          <label class="check wide"><input type="checkbox" v-model="enabled" /> 启用。停用后，绑在这个站点上的域名无法打开。</label>
          <label v-if="draft" class="field wide">
            <span>绑定域名，一行一个。第一行是主域名</span>
            <textarea class="hosts" v-model="form.hostsText" placeholder="example.com&#10;www.example.com"></textarea>
          </label>
        </div>
        <div class="row-actions" style="margin-top: 16px">
          <button class="btn primary" type="button" @click="save">保存站点</button>
          <button v-if="!draft" class="btn danger" type="button" @click="removeSite">删除站点</button>
          <span v-if="saved" class="msg ok">{{ saved }}</span>
        </div>
        <p v-if="error" class="msg error">{{ error }}</p>

        <div v-if="current && !draft" class="domain-box">
          <h2>域名绑定</h2>
          <table class="table">
            <thead>
              <tr><th>域名</th><th>角色</th><th></th></tr>
            </thead>
            <tbody>
              <tr v-for="domain in current.domains" :key="domain.id">
                <td>{{ domain.host }}</td>
                <td><span class="pill" :data-on="domain.is_primary">{{ domain.is_primary ? '主域名' : '别名' }}</span></td>
                <td class="row-actions">
                  <a class="btn small" :href="`/?host=${encodeURIComponent(domain.host)}`" target="_blank" rel="noopener">访问</a>
                  <button v-if="!domain.is_primary" class="btn small" type="button" @click="makePrimary(domain.id)">设为主域名</button>
                  <button class="btn small danger" type="button" @click="removeDomain(domain)">解除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div class="domain-add">
            <input v-model="newHost" placeholder="www.example.com" @keyup.enter="addDomain" />
            <button class="btn" type="button" @click="addDomain">绑定域名</button>
          </div>
        </div>
      </section>
      <section v-else class="card empty">还没有站点，点击右上角新建。</section>
    </div>
  </div>
</template>
