import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { capturePreviewHost } from './siteHost'
import './styles.css'

capturePreviewHost()

createApp(App).use(createPinia()).use(router).mount('#app')
