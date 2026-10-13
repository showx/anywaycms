<script setup>
import { reactive, ref, watch } from 'vue'
import { http } from '../../api'
import { useContextStore } from '../../stores/context'

const ctx = useContextStore()
const rows = ref([])
const error = ref('')
const saved = ref('')
const editing = ref(0)
const form = reactive(blank())

function blank() {
  return { name: '', slug: '', description: '', sort: 0 }
}

function edit(row) {
  editing.value = row.id
  Object.assign(form, {
    name: row.name,
    slug: row.slug,
    description: row.description,
    sort: row.sort,
  })
  error.value = ''
  saved.value = ''
}

function reset() {
  editing.value = 0
  Object.assign(form, blank())
}

async function load() {
  rows.value = []
  if (!ctx.siteId) return
  error.value = ''
  try {
    const res = await http.get(`/api/sites/${ctx.siteId}/categories`)
    rows.value = res.data || []
  } catch (err) {
    error.value = err.message
  }
}

async function save() {
  error.value = ''
  saved.value = ''
  try {
    if (editing.value) {
      await http.put(`/api/categories/${editing.value}`, form)
    } else {
      await http.post(`/api/sites/${ctx.siteId}/categories`, form)
    }
    reset()
    saved.value = '已保存'
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(row) {
  if (!window.confirm(`删除栏目「${row.name}」？`)) return
  error.value = ''
  try {
    await http.delete(`/api/categories/${row.id}`)
    if (editing.value === row.id) reset()
    await load()
  } catch (err) {
    error.value = err.message
  }
}

watch(() => ctx.siteId, () => {
  reset()
  load()
}, { immediate: true })
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>栏目</h1>
        <p class="muted">栏目只属于当前站点。换一个站点，栏目和文章都是另一套。</p>
      </div>
    </div>
    <p v-if="!ctx.siteId" class="card empty">还没有站点。<router-link to="/admin/sites">去创建</router-link></p>
    <div v-else class="split">
      <form class="card" @submit.prevent="save">
        <h2>{{ editing ? '编辑栏目' : '新建栏目' }}</h2>
        <label class="field"><span>名称</span><input v-model="form.name" required /></label>
        <label class="field"><span>别名</span><input v-model="form.slug" placeholder="可留空，保存后自动生成" /></label>
        <label class="field"><span>说明</span><textarea class="hosts" v-model="form.description"></textarea></label>
        <label class="field"><span>排序</span><input v-model.number="form.sort" type="number" /></label>
        <p v-if="error" class="msg error">{{ error }}</p>
        <p v-if="saved" class="msg ok">{{ saved }}</p>
        <div class="row-actions">
          <button class="btn primary" type="submit">保存</button>
          <button v-if="editing" class="btn" type="button" @click="reset">取消编辑</button>
        </div>
      </form>
      <div class="card">
        <table class="table" v-if="rows.length">
          <thead>
            <tr><th>栏目</th><th>别名</th><th>内容</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="row in rows" :key="row.id">
              <td>{{ row.name }}</td>
              <td>{{ row.slug }}</td>
              <td>{{ row.content_count }}</td>
              <td class="row-actions">
                <button class="btn small" type="button" @click="edit(row)">编辑</button>
                <button class="btn small danger" type="button" @click="remove(row)">删除</button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty">这个站点还没有栏目。</p>
      </div>
    </div>
  </div>
</template>
