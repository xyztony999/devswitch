<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NEmpty, NInput, NSelect, NSpin, useMessage } from 'naive-ui'
import BrandLogo from './BrandLogo.vue'
import { send, issueLabel } from '../bridge'
import { store } from '../store'
import type { Runtime } from '../types'

const message = useMessage()
const dlVersion = ref('')
const dlMirror = ref('official')
const mirrorOptions = [
  { label: '官方源', value: 'official' },
  { label: 'npmmirror', value: 'npmmirror' },
  { label: '清华源', value: 'tuna' }
]

const TOOLS = ['node', 'java', 'maven', 'gradle'] as const
const TOOL_LABELS: Record<string, string> = {
  node: 'Node.js',
  java: 'Java',
  maven: 'Maven',
  gradle: 'Gradle'
}
const TOOL_COMMANDS: Record<string, string> = {
  node: 'node / npm',
  java: 'java / javac',
  maven: 'mvn',
  gradle: 'gradle'
}
const TOOL_BLURBS: Record<string, string> = {
  node: 'JavaScript 运行时 · 版本切换对已打开的终端立即生效',
  java: 'JDK · 同步维护 JAVA_HOME（终端钩子 + 用户环境变量）',
  maven: '构建工具 · 同步维护 MAVEN_HOME',
  gradle: '构建工具 · 同步维护 GRADLE_HOME'
}

// 关于页是纯前端本地路由，不打扰后端 page 状态
const showAbout = ref(false)

const isToolPage = computed(() => TOOLS.includes(store.page as never))
const activeTool = computed<string>(() => (isToolPage.value ? store.page : 'node'))
const currentOf = (tool: string) => store.current[tool] || null

const isActive = (item: Runtime) => {
  const cur = store.current[item.tool]
  return !!(cur && cur.home === item.home)
}

const errorCount = computed(
  () => store.issues.filter((i) => i.level === 'error').length
)

watch(
  () => store.flash,
  (flash) => {
    if (!flash?.text) return
    if (flash.kind === 'error') message.error(flash.text)
    else message.success(flash.text)
  }
)

function setPage(page: string) {
  showAbout.value = false
  send({ op: 'page', page })
}

function download() {
  const version = dlVersion.value.trim()
  if (!version) {
    message.warning('请输入大版本号，例如 22 或 17')
    return
  }
  send({ op: 'install', version, mirror: dlMirror.value })
}

function useHome(home: string) {
  send({ op: 'use', home })
}
</script>

<template>
  <div class="app-shell">
    <!-- 顶栏：品牌 + 各工具当前版本（带 logo，点击直达对应页） -->
    <header class="app-topbar">
      <div class="app-brand">
        <span class="app-brand-mark">D</span>
        <span>DevSwitch</span>
      </div>
      <div class="topbar-current">
        <div
          v-for="tool in TOOLS"
          :key="tool"
          class="top-chip"
          :class="{ active: isToolPage && activeTool === tool }"
          @click="setPage(tool)"
        >
          <BrandLogo :kind="tool" size="sm" />
          <span class="top-chip-name">{{ tool }}</span>
          <span class="top-chip-ver">{{ currentOf(tool)?.version || '—' }}</span>
        </div>
      </div>
      <NButton size="small" secondary @click="send({ op: 'scan' })">重新扫描</NButton>
    </header>

    <div class="app-body">
      <!-- 左侧导航：常驻，任何页面都能一键切换（诊断不再困死） -->
      <nav class="app-nav">
        <div class="nav-section">运行时</div>
        <div
          v-for="tool in TOOLS"
          :key="tool"
          class="nav-item"
          :class="{ active: isToolPage && activeTool === tool }"
          @click="setPage(tool)"
        >
          <BrandLogo :kind="tool" size="lg" />
          <div class="nav-item-text">
            <div class="nav-item-name">{{ TOOL_LABELS[tool] }}</div>
            <div class="nav-item-sub">
              {{ currentOf(tool)?.version || '未选择' }}
            </div>
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
        </div>

        <div class="nav-spacer"></div>
        <div class="nav-footer">
          shim 目录
          <br /><code>{{ store.paths.localBin || '—' }}</code>
          <br />v{{ store.version || '2.0.0' }}
        </div>
      </nav>

      <!-- 主内容 -->
      <main class="app-main">
        <!-- 关于页 -->
        <template v-if="showAbout">
          <div class="tool-hero">
            <div class="tool-hero-icon">
              <span class="app-brand-mark" style="width: 34px; height: 34px; font-size: 17px">D</span>
            </div>
            <div>
              <div class="tool-hero-title">DevSwitch</div>
              <div class="tool-hero-sub">
                Node.js / Java / Maven / Gradle 用户级版本切换器 · v{{ store.version || '2.0.0' }}
              </div>
            </div>
          </div>
          <div class="ver-card" style="max-width: 560px">
            <div class="issue-msg" style="line-height: 1.8">
              扫描本机已有的运行时，收拢到一张列表里，一键切换；也可以直接下载安装指定大版本。
              切换发生在当前用户目录（shim + 环境变量 + shell 钩子），不需要管理员权限，
              已打开的终端立即生效。
            </div>
            <div class="ver-card-meta">
              <span>项目主页：https://github.com/xyztony999/devswitch</span>
              <span>开源协议：MIT</span>
            </div>
          </div>
        </template>

        <!-- 诊断页 -->
        <template v-else-if="store.page === 'doctor'">
          <div class="tool-hero">
            <div class="tool-hero-icon" style="font-size: 24px">🩺</div>
            <div style="flex: 1">
              <div class="tool-hero-title">
                环境诊断
                <span v-if="!store.issues.length" class="hero-current-tag">一切正常</span>
                <span v-else-if="errorCount" class="hero-current-tag" style="color: #ff8589; border-color: rgba(229,72,77,.4); background: rgba(229,72,77,.1)">
                  {{ errorCount }} 项需要处理
                </span>
                <span v-else class="hero-current-tag" style="color: #ffc75d; border-color: rgba(245,166,35,.4); background: rgba(245,166,35,.1)">
                  {{ store.issues.length }} 项建议关注
                </span>
              </div>
              <div class="tool-hero-sub">检查 PATH、shim、环境变量与 shell 钩子的冲突配置</div>
            </div>
            <NButton type="primary" @click="send({ op: 'fix' })">一键修复</NButton>
          </div>

          <NEmpty
            v-if="!store.issues.length"
            description="没有发现问题：PATH、shim 与 shell 钩子均无冲突"
            style="margin-top: 60px"
          />

          <div v-else class="ver-grid" style="grid-template-columns: repeat(auto-fill, minmax(360px, 1fr))">
            <div v-for="(issue, i) in store.issues" :key="issue.code + i" class="issue-card" :class="issue.level">
              <div class="issue-head">
                <span class="issue-level">{{ issueLabel(issue.level) }}</span>
                <span class="issue-code">{{ issue.code }}</span>
              </div>
              <div class="issue-msg">{{ issue.message }}</div>
            </div>
          </div>
        </template>

        <!-- 工具页 -->
        <template v-else>
          <div class="tool-hero">
            <div class="tool-hero-icon">
              <BrandLogo :kind="activeTool" size="xl" />
            </div>
            <div style="flex: 1; min-width: 0">
              <div class="tool-hero-title">
                {{ TOOL_LABELS[activeTool] }}
                <span v-if="currentOf(activeTool)" class="hero-current-tag">
                  {{ currentOf(activeTool)!.label || currentOf(activeTool)!.version }}
                </span>
              </div>
              <div class="tool-hero-sub">{{ TOOL_BLURBS[activeTool] }}</div>
            </div>
            <NButton size="small" secondary @click="send({ op: 'import' })">导入目录</NButton>
          </div>

          <!-- 下载条 -->
          <div class="dl-card">
            <span class="dl-label">下载{{ TOOL_LABELS[activeTool] }}大版本</span>
            <NInput
              v-model:value="dlVersion"
              size="small"
              placeholder="如 22 / 17 / 3"
              style="width: 130px"
              :disabled="store.installing"
              @keyup.enter="download"
            />
            <NSelect
              v-model:value="dlMirror"
              size="small"
              :options="mirrorOptions"
              style="width: 116px"
              :disabled="store.installing"
            />
            <NButton size="small" type="primary" ghost :loading="store.installing" @click="download">
              {{ store.installing ? '下载中…' : '下载并切换' }}
            </NButton>
          </div>

          <NSpin v-if="store.installing" size="small" style="width: 100%; padding: 4px 0 10px">
            <div class="issue-msg">正在下载，视网络可能需要几分钟，完成后自动切换。</div>
          </NSpin>

          <NEmpty
            v-if="!store.runtimes.length"
            :description="`本机没有发现 ${TOOL_LABELS[activeTool]}`"
            style="margin-top: 50px"
          >
            <template #extra>
              <NButton size="small" secondary @click="send({ op: 'scan' })">扫描本机</NButton>
              <NButton size="small" secondary @click="send({ op: 'import' })">导入目录</NButton>
            </template>
          </NEmpty>

          <div v-else class="ver-grid">
            <div
              v-for="item in store.runtimes"
              :key="item.home"
              class="ver-card"
              :class="{ active: isActive(item) }"
            >
              <div class="ver-card-head">
                <BrandLogo :kind="item.tool" size="lg" />
                <span class="ver-card-title">{{ item.label || item.version }}</span>
                <span v-if="isActive(item)" class="active-dot">当前</span>
              </div>
              <div class="ver-card-meta">
                <span>版本 <code>{{ item.version }}</code> · {{ item.vendor || '—' }}</span>
                <span><code>{{ item.home }}</code></span>
                <span v-if="item.binary"><code>{{ item.binary }}</code></span>
              </div>
              <div class="ver-card-actions">
                <NButton
                  v-if="!isActive(item)"
                  size="small"
                  type="primary"
                  @click="useHome(item.home)"
                >
                  切换到此版本
                </NButton>
                <NButton v-else size="small" disabled>已激活</NButton>
              </div>
            </div>
          </div>

          <div
            v-if="store.runtimes.length"
            class="issue-msg"
            style="margin-top: 14px; color: var(--text-faint)"
          >
            当前终端的 {{ TOOL_COMMANDS[activeTool] }} 已指向激活版本；切换后立即生效，无需重开终端。
          </div>
        </template>
      </main>
    </div>
  </div>
</template>
