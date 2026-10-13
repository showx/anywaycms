<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { http } from '../../api'
import { adminPath } from '../../adminPath'
import { useContextStore } from '../../stores/context'

const ctx = useContextStore()
const form = reactive({
  seo_title: '',
  description: '',
  keywords: '',
  url_mode: 'plain',
})
const indexable = ref(true)
const report = ref({ title: '', origin: '', primary_host: '', urls: [], gaps: [] })
const error = ref('')
const saved = ref('')

const previewTitle = computed(() => (form.seo_title || report.value.title || '网站标题').trim())
const previewDesc = computed(() => form.description || '这里会显示搜索结果里的描述。')
const previewHost = computed(() => report.value.origin || report.value.primary_host || '')

async function load() {
  error.value = ''
  saved.value = ''
  report.value = { title: '', origin: '', primary_host: '', urls: [], gaps: [] }
  if (!ctx.siteId) return
  const res = await http.get(`/api/sites/${ctx.siteId}/seo`)
  form.seo_title = res.data.seo_title || ''
  form.description = res.data.description || ''
  form.keywords = res.data.keywords || ''
  form.url_mode = res.data.url_mode === 'html' ? 'html' : 'plain'
  indexable.value = res.data.robots !== 'noindex'
  report.value = res.data
}

async function submit() {
  error.value = ''
  saved.value = ''
  try {
    const res = await http.put(`/api/sites/${ctx.siteId}/seo`, {
      seo_title: form.seo_title,
      description: form.description,
      keywords: form.keywords,
      robots: indexable.value ? 'index' : 'noindex',
      url_mode: form.url_mode,
    })
    form.seo_title = res.data.seo_title || ''
    form.description = res.data.description || ''
    form.keywords = res.data.keywords || ''
    form.url_mode = res.data.url_mode === 'html' ? 'html' : 'plain'
    indexable.value = res.data.robots !== 'noindex'
    report.value = res.data
    await ctx.loadSites(true)
    saved.value = '已保存。前台链接、标题和收录设置已更新。'
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
        <h1>SEO</h1>
        <p class="muted">标题、描述和收录按当前站点生效。别名域名会指向主域名，避免同一页被搜到两次。</p>
      </div>
    </div>
    <p v-if="!ctx.siteId" class="card empty">先选择或创建一个站点。</p>
    <template v-else>
      <form class="card" @submit.prevent="submit">
        <div class="snippet">
          <small>{{ previewHost }}</small>
          <strong>{{ previewTitle }}</strong>
          <p>{{ previewDesc }}</p>
        </div>
        <div class="form-grid">
          <label class="field wide">
            <span>首页标题</span>
            <input v-model="form.seo_title" maxlength="70" placeholder="留空则使用网站标题" />
          </label>
          <label class="field wide">
            <span>简介</span>
            <textarea class="hosts" v-model="form.description" maxlength="300"></textarea>
          </label>
          <label class="field wide">
            <span>关键词</span>
            <input v-model="form.keywords" maxlength="150" placeholder="用逗号分开" />
          </label>
          <label class="check wide"><input type="checkbox" v-model="indexable" /> 允许搜索引擎收录这个站点</label>
          <label class="field wide">
            <span>链接样式</span>
            <select v-model="form.url_mode">
              <option value="plain">动态：/article/别名</option>
              <option value="html">伪静态：/article/别名.html</option>
            </select>
          </label>
        </div>
        <p class="muted">简介也会出现在首页开头。关闭收录后，robots.txt 会拒绝整站。伪静态只改栏目、文章和单页的地址，两种写法都能打开，不在当前样式里的地址会跳到规范地址。</p>
        <p v-if="error" class="msg error">{{ error }}</p>
        <p v-if="saved" class="msg ok">{{ saved }}</p>
        <button class="btn primary" type="submit">保存 SEO</button>
      </form>

      <section class="card">
        <h2>站点地图</h2>
        <p class="muted">搜索引擎访问主域名下的 <code>/sitemap.xml</code> 和 <code>/robots.txt</code>。下面这些地址只包含当前站点已发布的首页、栏目、文章和单页。</p>
        <table class="table" v-if="report.urls?.length">
          <thead>
            <tr><th>路径</th><th>规范地址</th></tr>
          </thead>
          <tbody>
            <tr v-for="item in report.urls" :key="item.loc">
              <td>{{ item.path }}</td>
              <td>{{ item.loc }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty">还没有可提交的地址。</p>
      </section>

      <section class="card">
        <h2>待补充</h2>
        <table class="table" v-if="report.gaps?.length">
          <thead>
            <tr><th>页面</th><th>问题</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="gap in report.gaps" :key="gap.kind + gap.id">
              <td>{{ gap.title }}<div class="muted">{{ gap.path }}</div></td>
              <td>{{ gap.issue }}</td>
              <td>
                <router-link v-if="gap.kind === 'category'" :to="adminPath('/categories')">去填写</router-link>
                <router-link v-else :to="adminPath('/contents/' + gap.id)">去填写</router-link>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty">已发布的栏目和内容都有描述。</p>
      </section>
    </template>
  </div>
</template>
