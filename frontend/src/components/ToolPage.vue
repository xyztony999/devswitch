<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NCheckbox, NInput, NSelect, NSpin } from 'naive-ui'
import BrandLogo from './BrandLogo.vue'
import VersionCard from './VersionCard.vue'
import { send, sourceText } from '../bridge'
import { store } from '../store'
import { toolConfig, TOOL_CONFIGS } from '../tools'
import type { Runtime } from '../types'

const props = defineProps<{ toolId: string }>()
const emit = defineEmits<{ (e: 'ask-remove', item: Runtime): void }>()

const config = computed(() => toolConfig(props.toolId as never))
const current = computed(() => store.current[props.toolId] || null)

const dlVersion = ref('')
const dlMirror = ref('official')
watch(config, c => {
  if (!c.mirrors.some(m => m.value === dlMirror.value)) dlMirror.value = c.mirrors[0].value
})

function download() {
  const version = dlVersion.value.trim()
  if (!version && config.value.versionInput === 'major') {
    return
  }
  send({ op: 'install', version, mirror: dlMirror.value })
}
</script>

<template>
  <div>
    <div class="tool-hero">
      <div class="tool-hero-icon" :style="{ borderColor: config.accent + '55', background: config.accent + '14' }">
        <BrandLogo :kind="props.toolId" size="xl" />
      </div>
      <div style="flex: 1; min-width: 0">
        <div class="tool-hero-title">
          {{ config.label }}
          <span v-if="current" class="hero-current-tag" :style="{ borderColor: config.accent + '66', background: config.accent + '1f', color: config.accent }">
            {{ current.label || current.version }}
          </span>
        </div>
        <div class="tool-hero-sub">{{ config.blurb }}</div>
      </div>
      <NButton size="small" secondary @click="send({ op: 'import' })">导入目录</NButton>
    </div>

    <!-- 下载条：versionInput 决定是否显示版本输入（'none' 走 latest 通道） -->
    <div class="dl-card">
      <span class="dl-label">下载{{ config.label }}{{ config.versionInput === 'major' ? '大版本' : '' }}</span>
      <NInput
        v-if="config.versionInput === 'major'"
        v-model:value="dlVersion"
        size="small"
        placeholder="如 22 / 17 / 3"
        style="width: 130px"
        :disabled="store.installing"
        @keyup.enter="download"
      />
      <span v-else class="dl-label" style="color: var(--text-faint)">最新版</span>
      <NSelect
        v-model:value="dlMirror"
        size="small"
        :options="config.mirrors"
        style="width: 132px"
        :disabled="store.installing"
      />
      <NButton size="small" type="primary" ghost :loading="store.installing" @click="download">
        {{ store.installing ? '下载中…' : '下载并切换' }}
      </NButton>
    </div>

    <NSpin v-if="store.installing" size="small" style="width: 100%; padding: 4px 0 10px">
      <div class="issue-msg">正在下载，视网络可能需要几分钟，完成后自动切换。</div>
    </NSpin>

    <div v-if="!store.runtimes.length" style="margin-top: 50px; text-align: center" class="issue-msg">
      本机没有发现 {{ config.label }} ·
      <NButton size="small" secondary @click="send({ op: 'scan' })">扫描本机</NButton>
      <NButton size="small" secondary @click="send({ op: 'import' })">导入目录</NButton>
    </div>

    <div v-else class="ver-grid">
      <VersionCard
        v-for="item in store.runtimes"
        :key="item.home"
        :item="item"
        @ask-remove="emit('ask-remove', $event)"
      />
    </div>

    <div v-if="store.runtimes.length" class="issue-msg" style="margin-top: 14px; color: var(--text-faint)">
      当前终端的 {{ config.commands }} 已指向激活版本；切换后立即生效，无需重开终端。
    </div>
  </div>
</template>
