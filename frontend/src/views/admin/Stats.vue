<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { http } from '../../api'
import { useContextStore } from '../../stores/context'

const ctx = useContextStore()
const days = ref(7)
const siteId = ref(0)
const error = ref('')
const loading = ref(false)
const report = ref(null)
const ready = ref(false)

const maxViews = computed(() => {
  const list = report.value?.daily || []
  return Math.max(1, ...list.map((item) => item.views || 0))
})

function barHeight(views) {
  return `${Math.max(4, Math.round((views / maxViews.value) * 120))}px`
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await http.get('/api/stats', { params: { days: days.value, site_id: siteId.value } })
    report.value = res.data
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

function pickDays(value) {
  days.value = value
  load()
}

onMounted(async () => {
  await ctx.loadSites()
  siteId.value = ctx.siteId || 0
  ready.value = true
  await load()
})

watch(siteId, () => {
  if (!ready.value) return
  load()
})
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>访问统计</h1>
        <p class="muted">只统计前台页面。后台操作不计入。数字按站点分开，换域名看到的是另一套访问。</p>
      </div>
    </div>
    <div class="tabs">
      <button class="btn" :class="{ on: days === 7 }" type="button" @click="pickDays(7)">近 7 天</button>
      <button class="btn" :class="{ on: days === 30 }" type="button" @click="pickDays(30)">近 30 天</button>
      <button class="btn" :class="{ on: days === 90 }" type="button" @click="pickDays(90)">近 90 天</button>
      <select v-model.number="siteId">
        <option :value="0">全部站点</option>
        <option v-for="site in ctx.sites" :key="site.id" :value="site.id">{{ site.title }}</option>
      </select>
    </div>
    <p v-if="error" class="msg error">{{ error }}</p>
    <p v-else-if="loading" class="muted">正在汇总…</p>
    <template v-else-if="report">
      <div class="stat-grid">
        <article class="card stat"><strong>{{ report.summary.views }}</strong><span>浏览量</span></article>
        <article class="card stat"><strong>{{ report.summary.visitors }}</strong><span>独立访客</span></article>
        <article class="card stat"><strong>{{ report.summary.today_views }}</strong><span>今日浏览</span></article>
        <article class="card stat"><strong>{{ report.summary.today_visitors }}</strong><span>今日访客</span></article>
      </div>
      <section class="card">
        <h2>每日浏览</h2>
        <div class="bars" v-if="report.daily.length">
          <div v-for="day in report.daily" :key="day.date" class="bar-col" :title="`${day.date} ${day.views} 次`">
            <div class="bar" :style="{ height: barHeight(day.views) }"></div>
            <span v-if="days === 7">{{ day.date.slice(5) }}</span>
          </div>
        </div>
        <p v-else class="empty">这段时间还没有访问。</p>
      </section>
      <div class="split">
        <section class="card">
          <h2>热门页面</h2>
          <table class="table" v-if="report.paths.length">
            <thead><tr><th>路径</th><th>浏览</th></tr></thead>
            <tbody>
              <tr v-for="item in report.paths" :key="item.name">
                <td>{{ item.name }}</td>
                <td>{{ item.views }}</td>
              </tr>
            </tbody>
          </table>
          <p v-else class="empty">还没有页面访问。</p>
        </section>
        <section class="card">
          <h2>访问域名</h2>
          <table class="table" v-if="report.hosts.length">
            <thead><tr><th>域名</th><th>浏览</th></tr></thead>
            <tbody>
              <tr v-for="item in report.hosts" :key="item.name">
                <td>{{ item.name }}</td>
                <td>{{ item.views }}</td>
              </tr>
            </tbody>
          </table>
          <p v-else class="empty">还没有域名访问。</p>
        </section>
      </div>
      <section class="card" style="margin-top: 16px" v-if="!siteId && report.sites.length">
        <h2>各站点</h2>
        <table class="table">
          <thead><tr><th>站点</th><th>浏览</th></tr></thead>
          <tbody>
            <tr v-for="item in report.sites" :key="item.site_id">
              <td>{{ item.title }}</td>
              <td>{{ item.views }}</td>
            </tr>
          </tbody>
        </table>
      </section>
    </template>
  </div>
</template>
