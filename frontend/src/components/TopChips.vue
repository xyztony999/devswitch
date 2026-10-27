<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { NDropdown } from 'naive-ui'
import BrandLogo from './BrandLogo.vue'
import { TOOL_CONFIGS } from '../tools'
import { store } from '../store'
import type { Tool } from '../types'

const props = defineProps<{ activeTool: Tool }>()
const emit = defineEmits<{ (e: 'navigate', page: string): void }>()

// ≥1100px 全量 chips；以下折叠为单钮下拉（ResizeObserver 单点断点状态）
const wide = ref(true)
let ro: ResizeObserver | null = null
const host = ref<HTMLElement | null>(null)
onMounted(() => {
  ro = new ResizeObserver(entries => {
    wide.value = entries[0].contentRect.width >= 1100
  })
  if (host.value) ro.observe(host.value)
})
onBeforeUnmount(() => ro?.disconnect())

const currentOf = (id: string) => store.current[id] || null
const activeConfig = computed(() => TOOL_CONFIGS.find(t => t.id === props.activeTool))

const options = TOOL_CONFIGS.map(t => ({ label: `${t.label} ${currentOf(t.id)?.version || '—'}`, key: t.id }))
function onPick(key: string) {
  emit('navigate', key)
}
</script>

<template>
  <div ref="host" style="flex: 1; min-width: 0; overflow: hidden; display: flex; align-items: center">
    <!-- 宽：全量 chips -->
    <div v-if="wide" style="display: flex; align-items: center; gap: 8px; overflow: hidden">
      <div
        v-for="t in TOOL_CONFIGS"
        :key="t.id"
        class="top-chip"
        :class="{ active: activeTool === t.id }"
        @click="emit('navigate', t.id)"
      >
        <BrandLogo :kind="t.id" size="sm" />
        <span class="top-chip-name">{{ t.id }}</span>
        <span class="top-chip-ver">{{ currentOf(t.id)?.version || '—' }}</span>
      </div>
    </div>
    <!-- 窄：折叠单钮 -->
    <NDropdown
      v-else
      :options="options"
      trigger="click"
      @select="onPick"
    >
      <div class="top-chip" :class="{ active: true }" style="cursor: pointer">
        <BrandLogo :kind="activeTool" size="sm" />
        <span class="top-chip-name">{{ activeConfig?.label || activeTool }}</span>
        <span class="top-chip-ver">{{ currentOf(activeTool)?.version || '—' }}</span>
        <span style="font-size: 10px; color: var(--text-faint)">▾</span>
      </div>
    </NDropdown>
  </div>
</template>
