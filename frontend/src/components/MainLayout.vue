<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { NButton, NCheckbox, NModal } from 'naive-ui'
import { useMessage } from 'naive-ui'
import BrandLogo from './BrandLogo.vue'
import BrandMark from './BrandMark.vue'
import ToolPage from './ToolPage.vue'
import DoctorPage from './DoctorPage.vue'
import AboutPage from './AboutPage.vue'
import SideNav from './SideNav.vue'
import TopChips from './TopChips.vue'
import { send } from '../bridge'
import { store } from '../store'
import { TOOL_CONFIGS } from '../tools'
import type { Runtime, Tool } from '../types'

const message = useMessage()

// 关于页是纯前端本地路由，不打扰后端 page 状态
const showAbout = ref(false)

// 响应式断点（单点状态，resize 驱动）：
//   wide   ≥1100px：全量 chips + 常驻侧栏
//   mid    769–1099px：chips 折叠 + 图标栏侧栏
//   narrow ≤768px：chips 折叠 + 抽屉侧栏（汉堡开关）
const width = ref(window.innerWidth)
const onResize = () => (width.value = window.innerWidth)
onMounted(() => window.addEventListener('resize', onResize))
onBeforeUnmount(() => window.removeEventListener('resize', onResize))
const mode = computed<'wide' | 'mid' | 'narrow'>(() =>
  width.value <= 768 ? 'narrow' : width.value < 1100 ? 'mid' : 'wide'
)

const drawerOpen = ref(false)
const isToolPage = computed(() => TOOL_CONFIGS.some(t => t.id === store.page))
const activeTool = computed<Tool>(() => (isToolPage.value ? (store.page as Tool) : 'node'))

function navigate(page: string) {
  drawerOpen.value = false
  if (page === '__about') {
    showAbout.value = true
    return
  }
  showAbout.value = false
  send({ op: 'page', page })
}

// 移除确认层
const removeTarget = ref<Runtime | null>(null)
const removePurge = ref(false)
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
</script>

<template>
  <div class="app-shell">
    <header class="app-topbar">
      <div class="app-brand">
        <BrandMark :size="26" />
        <span>DevSwitch</span>
      </div>
      <TopChips :active-tool="activeTool" @navigate="navigate" />
      <NButton
        v-if="mode === 'narrow'"
        size="small"
        secondary
        aria-label="菜单"
        @click="drawerOpen = !drawerOpen"
      >
        ☰
      </NButton>
      <NButton size="small" secondary @click="send({ op: 'scan' })">重新扫描</NButton>
    </header>

    <div class="app-body">
      <SideNav
        :collapsed="mode === 'mid'"
        :drawer="mode === 'narrow'"
        :drawer-open="mode === 'narrow' && drawerOpen"
        @navigate="navigate"
        @close-drawer="drawerOpen = false"
      />

      <main class="app-main">
        <AboutPage v-if="showAbout" />
        <DoctorPage v-else-if="store.page === 'doctor'" />
        <ToolPage v-else :tool-id="activeTool" @ask-remove="askRemove" />
      </main>
    </div>

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
