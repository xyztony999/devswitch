<script setup lang="ts">
import { computed } from 'vue'
import { NButton } from 'naive-ui'
import BrandLogo from './BrandLogo.vue'
import { send, sourceText } from '../bridge'
import { store } from '../store'
import { toolConfig } from '../tools'
import type { Runtime } from '../types'

const props = defineProps<{ item: Runtime }>()
const emit = defineEmits<{ (e: 'ask-remove', item: Runtime): void }>()

const isActive = computed(() => {
  const cur = store.current[props.item.tool]
  return !!(cur && cur.home === props.item.home)
})

const purgeable = computed(() =>
  ['downloaded', 'imported', 'local'].includes(props.item.source)
)
const config = computed(() => toolConfig(props.item.tool))
</script>

<template>
  <div class="ver-card" :class="{ active: isActive }">
    <div class="ver-card-head">
      <BrandLogo :kind="item.tool" size="lg" />
      <span class="ver-card-title">{{ item.label || item.version }}</span>
      <span v-if="isActive" class="active-dot">当前</span>
    </div>
    <div class="ver-card-meta">
      <span>版本 <code>{{ item.version }}</code> · {{ item.vendor || sourceText(item.source) }}</span>
      <span><code>{{ item.home }}</code></span>
      <span v-if="item.binary"><code>{{ item.binary }}</code></span>
    </div>
    <div class="ver-card-actions">
      <NButton
        v-if="!isActive"
        size="small"
        type="primary"
        @click="send({ op: 'use', home: item.home })"
      >
        切换到此版本
      </NButton>
      <NButton v-else size="small" disabled>已激活</NButton>
      <NButton
        v-if="!isActive"
        size="small"
        quaternary
        type="error"
        @click="emit('ask-remove', item)"
      >
        移除
      </NButton>
    </div>
  </div>
</template>
