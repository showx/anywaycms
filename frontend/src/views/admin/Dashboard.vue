<script setup>
import { computed, ref, watch } from 'vue'
import { http } from '../../api'
import { useContextStore } from '../../stores/context'

const ctx = useContextStore()
const weekViews = ref(0)

const domainCount = computed(() => ctx.sites.reduce((sum, site) => sum + (site.domains?.length || 0), 0))
const articleCount = computed(() => ctx.sites.reduce((sum, site) => sum + (site.article_count || 0), 0))
const pageCount = computed(() => ctx.sites.reduce((sum, site) => sum + (site.page_count || 0), 0))

async function loadViews() {
  if (!ctx.siteId) {
    weekViews.value = 0
    return
  }
  try {
    const res = await http.get('/api/stats', { params: { days: 7, site_id: ctx.siteId } })
    weekViews.value = res.data.summary.views || 0
  } catch {
    weekViews.value = 0
  }
}

watch(() => ctx.siteId, loadViews, { immediate: true })

function primaryHost(site) {
  const domain = (site.domains || []).find((item) => item.is_primary) || (site.domains || [])[0]
  return domain?.host || '未绑定'
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>概览</h1>
        <p class="muted">一个站点就是一个网站。域名绑到哪个站点，打开这个域名就只看到那个站点的页面。</p>
      </div>
    </div>
    <div class="stat-grid">
      <article class="card stat"><strong>{{ ctx.sites.length }}</strong><span>站点</span></article>
      <article class="card stat"><strong>{{ domainCount }}</strong><span>域名</span></article>
      <article class="card stat"><strong>{{ articleCount }}</strong><span>文章</span></article>
      <article class="card stat"><strong>{{ pageCount }}</strong><span>单页</span></article>
      <article class="card stat">
        <strong>{{ weekViews }}</strong>
        <span>近 7 日访问</span>
        <router-link to="/admin/stats">查看统计</router-link>
      </article>
    </div>
    <div class="card">
      <table class="table" v-if="ctx.sites.length">
        <thead>
          <tr>
            <th>网站</th>
            <th>主域名</th>
            <th>文章 / 单页</th>
            <th>状态</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="site in ctx.sites" :key="site.id">
            <td>
              <strong>{{ site.title }}</strong>
              <div class="muted">{{ site.subtitle }}</div>
            </td>
            <td>{{ primaryHost(site) }}</td>
            <td>{{ site.article_count }} / {{ site.page_count }}</td>
            <td><span class="pill" :data-on="site.status === 1">{{ site.status === 1 ? '启用' : '停用' }}</span></td>
            <td class="row-actions">
              <a class="btn small" :href="`/?host=${encodeURIComponent(primaryHost(site))}`" target="_blank" rel="noopener">访问</a>
              <router-link class="btn small" to="/admin/sites">管理</router-link>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="empty">还没有站点。<router-link to="/admin/sites">创建第一个网站</router-link></p>
    </div>
  </div>
</template>
