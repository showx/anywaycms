<script setup>
import { reactive, ref } from 'vue'
import { http } from '../../api'

const form = reactive({ old: '', next: '', again: '' })
const error = ref('')
const saved = ref('')

async function submit() {
  error.value = ''
  saved.value = ''
  if (form.next !== form.again) {
    error.value = '两次输入的新密码不一致'
    return
  }
  try {
    await http.put('/api/auth/password', { old: form.old, new: form.next })
    form.old = ''
    form.next = ''
    form.again = ''
    saved.value = '密码已更新'
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>修改密码</h1>
        <p class="muted">这是整套站群共用的后台账号。</p>
      </div>
    </div>
    <form class="card narrow" @submit.prevent="submit">
      <label class="field"><span>原密码</span><input v-model="form.old" type="password" autocomplete="current-password" /></label>
      <label class="field"><span>新密码</span><input v-model="form.next" type="password" autocomplete="new-password" /></label>
      <label class="field"><span>再输入一次</span><input v-model="form.again" type="password" autocomplete="new-password" /></label>
      <p v-if="error" class="msg error">{{ error }}</p>
      <p v-if="saved" class="msg ok">{{ saved }}</p>
      <button class="btn primary" type="submit">更新密码</button>
    </form>
  </div>
</template>
