<script setup>
import { onMounted, reactive, ref } from 'vue'
import { http } from '../../api'

const form = reactive({
  system_name: '',
  system_notice: '',
  track_visits: 1,
  retain_days: 90,
  admin_path: '',
})
const error = ref('')
const saved = ref('')
const enabled = ref(true)
const loadedPath = ref('')

async function load() {
  error.value = ''
  const res = await http.get('/api/settings')
  Object.assign(form, res.data)
  loadedPath.value = res.data.admin_path || ''
  enabled.value = res.data.track_visits === 1
}

async function submit() {
  error.value = ''
  saved.value = ''
  const previous = loadedPath.value
  try {
    const res = await http.put('/api/settings', {
      system_name: form.system_name,
      system_notice: form.system_notice,
      track_visits: enabled.value ? 1 : 0,
      retain_days: Number(form.retain_days),
      admin_path: form.admin_path,
    })
    const next = res.data.admin_path || ''
    if (next && next !== previous) {
      window.location.replace(`/${next}/settings`)
      return
    }
    Object.assign(form, res.data)
    loadedPath.value = next
    enabled.value = res.data.track_visits === 1
    saved.value = '已保存。刷新后侧栏名称会更新。'
  } catch (err) {
    error.value = err.message
  }
}

onMounted(() => {
  load().catch((err) => {
    error.value = err.message
  })
})
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>系统设置</h1>
        <p class="muted">这些设置属于整套后台，不随当前站点变化。</p>
      </div>
    </div>
    <form class="card narrow" @submit.prevent="submit">
      <label class="field"><span>系统名称</span><input v-model="form.system_name" required /></label>
      <label class="field"><span>侧栏说明</span><input v-model="form.system_notice" /></label>
      <label class="check"><input type="checkbox" v-model="enabled" /> 记录前台访问</label>
      <label class="field"><span>统计保留天数</span><input v-model.number="form.retain_days" type="number" min="7" max="365" /></label>
      <label class="field"><span>后台路径</span><input v-model="form.admin_path" required maxlength="32" autocomplete="off" spellcheck="false" /></label>
      <p class="muted">关闭记录后，新的访问不再写入。超过保留天数的旧记录会在下一次访问时清掉。</p>
      <p class="muted">后台地址是 /你设置的路径。只能用小写字母和数字，3 到 32 位，以字母开头，不能使用 admin。改完会立刻跳到新地址，请记下来。</p>
      <p v-if="error" class="msg error">{{ error }}</p>
      <p v-if="saved" class="msg ok">{{ saved }}</p>
      <button class="btn primary" type="submit">保存设置</button>
    </form>
  </div>
</template>
