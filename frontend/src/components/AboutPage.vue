<script setup lang="ts">
import { NButton } from 'naive-ui'
import BrandMark from './BrandMark.vue'
import { send } from '../bridge'
import { store } from '../store'

const RELEASE_URL = 'https://github.com/xyztony999/devswitch/releases/latest'
</script>

<template>
  <div>
    <div class="tool-hero">
      <div class="tool-hero-icon">
        <BrandMark :size="34" />
      </div>
      <div>
        <div class="tool-hero-title">DevSwitch</div>
        <div class="tool-hero-sub">
          Node.js / Java / Maven / Gradle / Android SDK / Go / Python 用户级版本切换器 ·
          v{{ store.version || '—' }}
        </div>
      </div>
    </div>

    <div v-if="store.updateAvailable" class="update-banner">
      <div style="flex: 1; min-width: 0">
        <div style="font-weight: 700; margin-bottom: 2px">发现新版本 v{{ store.latest }}</div>
        <div class="issue-msg">
          当前 v{{ store.version }} · 建议下载新版安装包覆盖安装，版本选择与运行时数据不受影响
        </div>
      </div>
      <NButton size="small" type="primary" @click="send({ op: 'open-url', url: RELEASE_URL })">
        打开发布页
      </NButton>
    </div>

    <div class="ver-card" style="max-width: 560px">
      <div class="ver-card-meta" style="flex-direction: row; justify-content: space-between">
        <span>当前版本 <code>v{{ store.version || '—' }}</code></span>
        <span>
          最新版本
          <code v-if="store.latest">v{{ store.latest }}</code>
          <code v-else>未检查</code>
          <template v-if="store.latest && !store.updateAvailable">（已是最新）</template>
        </span>
      </div>
      <div class="ver-card-actions">
        <NButton
          size="small"
          secondary
          :loading="store.updateChecking"
          @click="send({ op: 'check-update' })"
        >
          检查更新
        </NButton>
      </div>
    </div>

    <div class="ver-card" style="max-width: 560px">
      <div class="issue-msg" style="line-height: 1.8">
        扫描本机已有的运行时，收拢到一张列表里，一键切换；也可以直接下载安装指定大版本。
        切换发生在当前用户目录（shim + 环境变量 + shell 钩子），不需要管理员权限，
        已打开的终端立即生效。项目环境自动切换：目录内放置 .devswitch 或 .nvmrc。
      </div>
      <div class="ver-card-meta">
        <span>项目主页：https://github.com/xyztony999/devswitch</span>
        <span>开源协议：MIT</span>
      </div>
    </div>
  </div>
</template>
