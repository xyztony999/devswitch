import { createApp } from 'vue'
import App from './App.vue'
import { store } from './store'
import type { AppState } from './types'
import './styles/global.css'

window.__setState = (next: AppState) => {
  Object.assign(store, next)
}

createApp(App).mount('#app')
