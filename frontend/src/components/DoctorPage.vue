<script setup lang="ts">
import { computed } from 'vue'
import { NButton } from 'naive-ui'
import { send, issueLabel } from '../bridge'
import { store } from '../store'

const errorCount = computed(() => store.issues.filter(i => i.level === 'error').length)
</script>

<template>
  <div>
    <div class="tool-hero">
      <div class="tool-hero-icon" style="font-size: 24px">🩺</div>
      <div style="flex: 1">
        <div class="tool-hero-title">
          环境诊断
          <span v-if="!store.issues.length" class="hero-current-tag">一切正常</span>
          <span
            v-else-if="errorCount"
            class="hero-current-tag"
            style="color: #ff8589; border-color: rgba(229,72,77,.4); background: rgba(229,72,77,.1)"
          >
            {{ errorCount }} 项需要处理
          </span>
          <span
            v-else
            class="hero-current-tag"
            style="color: #ffc75d; border-color: rgba(245,166,35,.4); background: rgba(245,166,35,.1)"
          >
            {{ store.issues.length }} 项建议关注
          </span>
        </div>
        <div class="tool-hero-sub">检查 PATH、shim、环境变量与 shell 钩子的冲突配置</div>
      </div>
      <NButton type="primary" @click="send({ op: 'fix' })">一键修复</NButton>
    </div>

    <div v-if="!store.issues.length" class="issue-msg" style="margin-top: 60px; text-align: center">
      没有发现问题：PATH、shim 与 shell 钩子均无冲突
    </div>

    <div v-else class="ver-grid" style="grid-template-columns: repeat(auto-fill, minmax(360px, 1fr))">
      <div
        v-for="(issue, i) in store.issues"
        :key="issue.code + i"
        class="issue-card"
        :class="issue.level"
      >
        <div class="issue-head">
          <span class="issue-level">{{ issueLabel(issue.level) }}</span>
          <span class="issue-code">{{ issue.code }}</span>
        </div>
        <div class="issue-msg">{{ issue.message }}</div>
      </div>
    </div>
  </div>
</template>
