<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { http } from '../api'
import { useAuthStore } from '../stores/auth'
import { useContextStore } from '../stores/context'

const router = useRouter()
const auth = useAuthStore()
const ctx = useContextStore()
const systemName = ref('站群 CMS')
const systemNotice = ref('一套后台，多个网站')
ctx.loadSites().catch(() => {})
onMounted(async () => {
  try {
    const res = await http.get('/api/system')
    systemName.value = res.data.system_name || systemName.value
    systemNotice.value = res.data.system_notice || systemNotice.value
  } catch {
    // 名称读取失败时继续使用默认文案。
  }
})

function onSite(event) {
  ctx.setSite(Number(event.target.value))
}

function logout() {
  auth.logout()
  router.push('/admin/login')
}
</script>

<template>
  <div class="admin-shell">
    <aside class="sidebar">
      <div class="brand">
        <strong>{{ systemName }}</strong>
        <small>{{ systemNotice }}</small>
      </div>
      <nav class="side-nav">
        <router-link to="/admin" exact-active-class="on">概览</router-link>
        <router-link to="/admin/sites" active-class="on">站点与域名</router-link>
        <router-link to="/admin/categories" active-class="on">栏目</router-link>
        <router-link to="/admin/contents" active-class="on">内容</router-link>
        <router-link to="/admin/menus" active-class="on">菜单</router-link>
        <router-link to="/admin/stats" active-class="on">访问统计</router-link>
        <router-link to="/admin/settings" active-class="on">系统设置</router-link>
        <router-link to="/admin/password" active-class="on">修改密码</router-link>
      </nav>
    </aside>
    <div class="content">
      <header class="topbar">
        <label v-if="ctx.sites.length" class="site-switch">
          当前站点
          <select :value="ctx.siteId" @change="onSite">
            <option v-for="site in ctx.sites" :key="site.id" :value="site.id">{{ site.title }}</option>
          </select>
        </label>
        <router-link v-else to="/admin/sites">先创建站点</router-link>
        <span class="grow"></span>
        <span class="muted">{{ auth.username }}</span>
        <button class="btn small" type="button" @click="logout">退出</button>
      </header>
      <div class="content-inner">
        <router-view />
      </div>
    </div>
  </div>
</template>
