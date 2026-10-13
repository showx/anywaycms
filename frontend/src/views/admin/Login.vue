<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { http } from '../../api'
import { useAuthStore } from '../../stores/auth'
import { adminPath, isAdminLocation } from '../../adminPath'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const username = ref('admin')
const password = ref('')
const error = ref('')
const loading = ref(false)
const systemName = ref('站群 CMS')
const systemNotice = ref('一套后台，多个网站')

const redirect = computed(() => {
  const value = route.query.redirect
  const loginPath = adminPath('/login')
  const home = adminPath('') || '/'
  if (typeof value !== 'string' || !isAdminLocation(value.split('?')[0]) || value === loginPath || value.startsWith(`${loginPath}?`)) return home
  return value
})

onMounted(async () => {
  try {
    const res = await http.get('/api/system')
    systemName.value = res.data.system_name || systemName.value
    systemNotice.value = res.data.system_notice || systemNotice.value
  } catch {
    // 保持默认名称。
  }
  if (auth.token) router.replace(redirect.value)
})

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value.trim(), password.value)
    router.push(redirect.value)
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login">
    <section class="login-intro">
      <p class="eyebrow">{{ systemName }}</p>
      <h1>{{ systemNotice }}</h1>
      <p>按域名识别站点。同一个系统里维护多家企业站，访客打开哪个域名，就看到哪一家的栏目、文章和单页。</p>
      <p>演示数据里，本机打开是星澜科技；带上 <code>?host=qinghe.local</code> 就是青禾文旅。</p>
    </section>
    <form class="login-card" @submit.prevent="submit">
      <h2>登录后台</h2>
      <label class="field">
        <span>账号</span>
        <input v-model="username" autocomplete="username" />
      </label>
      <label class="field">
        <span>密码</span>
        <input v-model="password" type="password" autocomplete="current-password" />
      </label>
      <p v-if="error" class="msg error">{{ error }}</p>
      <button class="btn primary" type="submit" :disabled="loading">{{ loading ? '登录中…' : '进入后台' }}</button>
      <p class="muted">首次启动账号 admin，密码 admin123。登录后请到「修改密码」换掉它。</p>
    </form>
  </div>
</template>
