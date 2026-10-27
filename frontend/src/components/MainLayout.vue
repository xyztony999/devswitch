<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NCheckbox, NModal } from 'naive-ui'
import { useMessage } from 'naive-ui'
import BrandLogo from './BrandLogo.vue'
import BrandMark from './BrandMark.vue'
import ToolPage from './ToolPage.vue'
import DoctorPage from './DoctorPage.vue'
import AboutPage from './AboutPage.vue'
import { send } from '../bridge'
import { store } from '../store'
import { TOOL_CONFIGS } from '../tools'
import type { Runtime, Tool } from '../types'

const message = useMessage()

// 关于页是纯前端本地路由，不打扰后端 page 状态
const showAbout = ref(false)

const isToolPage = computed(() => TOOL_CONFIGS.some(t => t.id === store.page))
const activeTool = computed<Tool>(() => (isToolPage.value ? (store.page as Tool) : 'node'))
const currentOf = (id: string) => store.current[id] || null

// 移除确认层
const removeTarget = ref<Runtime | null>(null)
const removePurge = ref(false)
// 外来安装（system/env/path）不显示删盘选项；目录是否在用户主目录内由后端红线判定
function canPurgeItem(item: Runtime): boolean {
  return ['downloaded', 'imported', 'local'].includes(item.source)
}
function askRemove(item: Runtime) {
  removeTarget.value = item
  removePurge.value = false
}
function confirmRemove() {
  const item = removeTarget.value
  if (!item) return
  send({ op: 'remove', home: item.home, purge: removePurge.value, confirmed: true })
  removeTarget.value = null
}

watch(
  () => store.flash,
  flash => {
    if (!flash?.text) return
    if (flash.kind === 'error') message.error(flash.text)
    else message.success(flash.text)
  }
)

function setPage(page: string) {
  showAbout.value = false
  send({ op: 'page', page })
}
</script>

<template>
  <div class="app-shell">
    <!-- 顶栏：品牌 + 各工具当前版本（带 logo，点击直达对应页） -->
    <header class="app-topbar">
      <div class="app-brand">
        <BrandMark :size="26" />
        <span>DevSwitch</span>
      </div>
      <div class="topbar-current" style="flex-wrap: wrap; row-gap: 4px">
        <div
          v-for="t in TOOL_CONFIGS"
          :key="t.id"
          class="top-chip"
          :class="{ active: isToolPage && activeTool === t.id }"
          @click="setPage(t.id)"
        >
          <BrandLogo :kind="t.id" size="sm" />
          <span class="top-chip-name">{{ t.id }}</span>
          <span class="top-chip-ver">{{ currentOf(t.id)?.version || '—' }}</span>
        </div>
      </div>
      <NButton size="small" secondary @click="send({ op: 'scan' })">重新扫描</NButton>
    </header>

    <div class="app-body">
      <!-- 左侧导航：常驻，任何页面都能一键切换 -->
      <nav class="app-nav">
        <div class="nav-section">运行时</div>
        <div
          v-for="t in TOOL_CONFIGS"
          :key="t.id"
          class="nav-item"
          :class="{ active: isToolPage && activeTool === t.id }"
          @click="setPage(t.id)"
        >
          <BrandLogo :kind="t.id" size="lg" />
          <div class="nav-item-text">
            <div class="nav-item-name">{{ t.label }}</div>
            <div class="nav-item-sub">{{ currentOf(t.id)?.version || '未选择' }}</div>
          </div>
        </div>

        <div class="nav-section">工具</div>
        <div
          class="nav-item"
          :class="{ active: store.page === 'doctor' && !showAbout }"
          @click="setPage('doctor')"
        >
          <span style="font-size: 18px; width: 22px; text-align: center">🩺</span>
          <div class="nav-item-text">
            <div class="nav-item-name">环境诊断</div>
            <div class="nav-item-sub">PATH / shim / 钩子</div>
          </div>
          <span v-if="store.issues.length" class="nav-badge">{{ store.issues.length }}</span>
          <span v-else class="nav-badge ok">✓</span>
        </div>
        <div class="nav-item" :class="{ active: showAbout }" @click="showAbout = true">
          <span style="font-size: 18px; width: 22px; text-align: center">ℹ️</span>
          <div class="nav-item-text">
            <div class="nav-item-name">关于</div>
          </div>
          <span v-if="store.updateAvailable" class="nav-badge ok">NEW</span>
        </div>

        <div class="nav-spacer"></div>
        <div class="nav-footer">
          shim 目录
          <br /><code>{{ store.paths.localBin || '—' }}</code>
          <br />v{{ store.version || '—' }}
        </div>
      </nav>

      <!-- 主内容：路由到子页 -->
      <main class="app-main">
        <AboutPage v-if="showAbout" />
        <DoctorPage v-else-if="store.page === 'doctor'" />
        <ToolPage v-else :tool-id="activeTool" @ask-remove="askRemove" />
      </main>
    </div>

    <!-- 移除确认层 -->
    <NModal
      :show="!!removeTarget"
      preset="dialog"
      :title="removeTarget ? `移除 ${removeTarget.label || removeTarget.version}` : ''"
      positive-text="确认移除"
      negative-text="取消"
      :type="removePurge ? 'error' : 'warning'"
      @positive-click="confirmRemove"
      @negative-click="removeTarget = null"
      @close="removeTarget = null"
    >
      <div v-if="removeTarget" style="line-height: 1.8">
        <div><code>{{ removeTarget.home }}</code></div>
        <div class="issue-msg" v-if="!removePurge">仅从 DevSwitch 清单移除，不删除磁盘文件。</div>
        <div class="issue-msg" v-else style="color: #ff8589">将永久删除上述目录。</div>
        <NCheckbox
          v-if="canPurgeItem(removeTarget)"
          :checked="removePurge"
          style="margin-top: 8px"
          @update:checked="(v: boolean) => (removePurge = v)"
        >
          同时删除磁盘目录（不可恢复）
        </NCheckbox>
      </div>
    </NModal>
  </div>
</template>
