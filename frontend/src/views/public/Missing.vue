<script setup>
import { usePublicStore } from '../../stores/public'

const store = usePublicStore()
</script>

<template>
  <section class="missing">
    <p class="eyebrow">无法打开网站</p>
    <h1>{{ store.error || '这个域名还没有绑定站点' }}</h1>
    <p>当前域名：{{ store.host || '未知' }}</p>
    <p>到后台新建一个站点，或把这个域名绑到已有站点。绑定之后，用这个域名访问就会看到对应网站的页面。</p>
    <router-link class="btn primary" to="/admin">进入后台</router-link>
  </section>
</template>
