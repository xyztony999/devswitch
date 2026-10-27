<script setup lang="ts">
import BrandLogo from './BrandLogo.vue'
import { TOOL_CONFIGS } from '../tools'
import { store } from '../store'

// 左侧导航：常驻 / 图标栏（中宽 collapsed）/ 抽屉（窄宽 drawer 模式）
defineProps<{
  collapsed?: boolean // 图标栏模式（只显 logo + 版本角标）
  drawer?: boolean // 抽屉模式（fixed 滑入）
  drawerOpen?: boolean
}>()
const emit = defineEmits<{
  (e: 'navigate', page: string): void
  (e: 'close-drawer'): void
}>()
</script>

<template>
  <div
    class="side-nav-hit"
    v-if="drawer && drawerOpen"
    @click="emit('close-drawer')"
  ></div>
  <nav
    class="app-nav"
    :class="{
      'is-iconbar': collapsed,
      'is-drawer': drawer,
      'is-drawer-open': drawer && drawerOpen
    }"
  >
    <div class="nav-section" v-if="!collapsed">运行时</div>
    <div
      v-for="t in TOOL_CONFIGS"
      :key="t.id"
      class="nav-item"
      :class="{ active: store.page === t.id }"
      :title="collapsed ? t.label : undefined"
      @click="emit('navigate', t.id)"
    >
      <BrandLogo :kind="t.id" size="lg" />
      <div class="nav-item-text" v-if="!collapsed">
        <div class="nav-item-name">{{ t.label }}</div>
        <div class="nav-item-sub">{{ store.current[t.id]?.version || '未选择' }}</div>
      </div>
    </div>

    <div class="nav-section" v-if="!collapsed">工具</div>
    <div
      class="nav-item"
      :class="{ active: store.page === 'doctor' }"
      :title="collapsed ? '环境诊断' : undefined"
      @click="emit('navigate', 'doctor')"
    >
      <span style="font-size: 18px; width: 22px; text-align: center">🩺</span>
      <div class="nav-item-text" v-if="!collapsed">
        <div class="nav-item-name">环境诊断</div>
        <div class="nav-item-sub">PATH / shim / 钩子</div>
      </div>
      <span v-if="store.issues.length && !collapsed" class="nav-badge">{{ store.issues.length }}</span>
      <span v-else-if="!collapsed" class="nav-badge ok">✓</span>
    </div>
    <div
      class="nav-item"
      :title="collapsed ? '关于' : undefined"
      @click="emit('navigate', '__about')"
    >
      <span style="font-size: 18px; width: 22px; text-align: center">ℹ️</span>
      <div class="nav-item-text" v-if="!collapsed">
        <div class="nav-item-name">关于</div>
      </div>
      <span v-if="store.updateAvailable" class="nav-badge ok">NEW</span>
    </div>

    <div class="nav-spacer"></div>
    <div class="nav-footer" v-if="!collapsed">
      shim 目录
      <br /><code>{{ store.paths.localBin || '—' }}</code>
      <br />v{{ store.version || '—' }}
    </div>
  </nav>
</template>

<style>
.side-nav-hit {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 90;
}
.app-nav.is-iconbar {
  width: 64px;
  padding: 10px 6px;
}
.app-nav.is-iconbar .nav-item {
  justify-content: center;
  padding: 8px 4px;
}
.app-nav.is-drawer {
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  transform: translateX(-100%);
  transition: transform 0.18s ease;
  box-shadow: 0 0 24px rgba(0, 0, 0, 0.5);
}
.app-nav.is-drawer-open {
  transform: translateX(0);
}
</style>
