import { reactive } from 'vue'
import type { AppState } from './types'

export const store = reactive<AppState>({
  page: 'node',
  version: '',
  current: {},
  runtimes: [],
  issues: [],
  selected: '',
  flash: null,
  paths: {}
})
