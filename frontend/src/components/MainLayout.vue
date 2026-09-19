<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NButton,
  NCard,
  NEmpty,
  NLayout,
  NLayoutContent,
  NLayoutFooter,
  NLayoutHeader,
  NLayoutSider,
  NModal,
  NSpace,
  NTabPane,
  NTabs,
  NTag,
  NText,
  useMessage
} from 'naive-ui'
import BrandLogo from './BrandLogo.vue'
import { send, sourceText, issueLabel } from '../bridge'
import { store } from '../store'
import type { Runtime } from '../types'

const message = useMessage()
const showAbout = ref(false)

const current = computed(() => {
  if (store.page === 'doctor') return null
  return store.current[store.page] || null
})

const selectedRuntime = computed<Runtime | null>(() => {
  return store.runtimes.find((item) => item.home === store.selected) || store.runtimes[0] || null
})

const selectedIssue = computed(() => {
  const index = Number(store.selected) || 0
  return store.issues[index] || store.issues[0] || null
})

const isActive = (item: Runtime) => {
  const cur = store.current[item.tool]
  return !!(cur && cur.home === item.home)
}

watch(
  () => store.flash,
  (flash) => {
    if (!flash?.text) return
    if (flash.kind === 'error') message.error(flash.text)
    else message.success(flash.text)
  }
)

function setPage(page: string) {
  send({ op: 'page', page })
}

function selectHome(home: string) {
  send({ op: 'select', home })
}

function useHome(home: string) {
  send({ op: 'use', home })
}
</script>

<template>
  <NLayout style="height: 100vh" position="absolute">
    <NLayoutHeader
      bordered
      style="height: 56px; padding: 0 16px; display: flex; align-items: center"
    >
      <div style="display: flex; width: 100%; align-items: center; gap: 16px">
        <NText strong style="font-size: 16px">DevSwitch</NText>
        <NTabs
          type="segment"
          size="small"
          :value="store.page === 'doctor' ? undefined : store.page"
          style="max-width: 260px"
          @update:value="setPage"
        >
          <NTabPane name="node">
            <template #tab>
              <span style="display: inline-flex; align-items: center; gap: 6px">
                <BrandLogo kind="node" />
                Node.js
              </span>
            </template>
          </NTabPane>
          <NTabPane name="java">
            <template #tab>
              <span style="display: inline-flex; align-items: center; gap: 6px">
                <BrandLogo kind="java" />
                Java
              </span>
            </template>
          </NTabPane>
        </NTabs>

        <div style="flex: 1; display: flex; align-items: center; gap: 8px; min-width: 0">
          <NText depth="3">当前激活：</NText>
          <NTag
            v-if="store.page === 'doctor'"
            :type="store.issues.length ? 'warning' : 'success'"
            size="small"
            round
          >
            {{ store.issues.length ? store.issues.length + ' 项' : '正常' }}
          </NTag>
          <NTag v-else-if="current" type="success" size="small" round>{{ current.version }}</NTag>
          <NTag v-else type="warning" size="small" round>未选择</NTag>
        </div>

        <NSpace>
          <NButton size="small" @click="send({ op: 'scan' })">重新扫描</NButton>
          <NButton v-if="store.page !== 'doctor'" size="small" @click="send({ op: 'import' })">
            导入
          </NButton>
          <NButton
            size="small"
            :type="store.page === 'doctor' ? 'primary' : 'default'"
            @click="setPage('doctor')"
          >
            诊断
          </NButton>
          <NButton size="small" quaternary @click="showAbout = true">关于</NButton>
        </NSpace>
      </div>
    </NLayoutHeader>

    <NLayout has-sider position="absolute" style="top: 56px; bottom: 36px">
      <NLayoutSider bordered :width="320" :native-scrollbar="false" content-style="padding: 12px">
        <div style="display: flex; flex-direction: column; gap: 8px">
          <template v-if="store.page === 'doctor'">
            <NEmpty v-if="!store.issues.length" description="没有发现问题" style="margin-top: 40px" />
            <NCard
              v-for="(issue, index) in store.issues"
              :key="issue.code + index"
              size="small"
              hoverable
              class="runtime-card"
              :class="{ selected: String(index) === String(store.selected) }"
              style="cursor: pointer"
              @click="selectHome(String(index))"
            >
              <NText strong>{{ issueLabel(issue.level) }}</NText>
              <NText depth="3" style="display: block; font-size: 12px; margin-top: 4px">
                {{ issue.message }}
              </NText>
            </NCard>
          </template>
          <template v-else>
            <NEmpty
              v-if="!store.runtimes.length"
              :description="store.page === 'node' ? '本机没有发现 Node.js' : '本机没有发现 Java'"
              style="margin-top: 40px"
            >
              <template #extra>
                <NSpace>
                  <NButton size="small" @click="send({ op: 'scan' })">扫描本机</NButton>
                  <NButton size="small" @click="send({ op: 'import' })">导入目录</NButton>
                </NSpace>
              </template>
            </NEmpty>
            <NCard
              v-for="item in store.runtimes"
              :key="item.home"
              size="small"
              hoverable
              class="runtime-card"
              :class="{ selected: item.home === store.selected }"
              style="cursor: pointer"
              @click="selectHome(item.home)"
            >
              <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 4px">
                <BrandLogo :kind="item.tool" large />
                <NText strong style="font-size: 14px">{{ item.label || item.version }}</NText>
                <NTag v-if="isActive(item)" size="tiny" type="success" round>激活</NTag>
              </div>
              <NText depth="3" style="font-size: 12px; display: block; margin-bottom: 8px">
                {{ sourceText(item.source) }} · {{ item.home }}
              </NText>
              <NSpace size="small">
                <NButton v-if="!isActive(item)" size="tiny" type="primary" @click.stop="useHome(item.home)">
                  切换
                </NButton>
                <NButton size="tiny" quaternary @click.stop="selectHome(item.home)">详情</NButton>
              </NSpace>
            </NCard>
          </template>
        </div>
      </NLayoutSider>

      <NLayoutContent :native-scrollbar="false" content-style="padding: 16px 20px">
        <NCard v-if="store.page === 'doctor'" size="small" style="max-width: 720px">
          <NText strong style="font-size: 16px; display: block; margin-bottom: 16px">PATH 与冲突</NText>
          <NText depth="3" style="display: block; margin-bottom: 6px">说明</NText>
          <NText style="display: block; margin-bottom: 16px">
            {{ selectedIssue ? selectedIssue.message : '检查 shim、JAVA_HOME，以及 shell 配置里手写的 Node 路径。' }}
          </NText>
          <NButton type="primary" @click="send({ op: 'fix' })">立即修复</NButton>
        </NCard>

        <NEmpty
          v-else-if="!selectedRuntime"
          :description="
            store.runtimes.length
              ? '从左侧选择一个版本，或点右上角「重新扫描」'
              : '安装后点重新扫描，或导入安装目录'
          "
        />

        <NCard v-else size="small" style="max-width: 720px">
          <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 16px">
            <BrandLogo :kind="selectedRuntime.tool" large />
            <NText strong style="font-size: 16px">{{ selectedRuntime.label || selectedRuntime.version }}</NText>
            <NTag v-if="isActive(selectedRuntime)" size="tiny" type="success" round>激活</NTag>
          </div>
          <NText depth="3" style="display: block; margin-bottom: 4px">版本</NText>
          <NText style="display: block; margin-bottom: 12px">{{ selectedRuntime.version }}</NText>
          <NText depth="3" style="display: block; margin-bottom: 4px">安装目录</NText>
          <NText style="display: block; margin-bottom: 12px"><code>{{ selectedRuntime.home }}</code></NText>
          <NText depth="3" style="display: block; margin-bottom: 4px">可执行文件</NText>
          <NText style="display: block; margin-bottom: 12px"><code>{{ selectedRuntime.binary }}</code></NText>
          <NText depth="3" style="display: block; margin-bottom: 4px">来源</NText>
          <NText style="display: block; margin-bottom: 16px">{{ sourceText(selectedRuntime.source) }}</NText>
          <NText v-if="isActive(selectedRuntime)" depth="3">
            当前终端的 {{ selectedRuntime.tool === 'java' ? 'java / javac' : 'node / npm' }} 已指向这个版本。
          </NText>
          <NButton v-else type="primary" @click="useHome(selectedRuntime.home)">切换到此版本</NButton>
        </NCard>
      </NLayoutContent>
    </NLayout>

    <NLayoutFooter
      bordered
      position="absolute"
      style="height: 36px; padding: 0 16px; display: flex; align-items: center; gap: 16px"
    >
      <NText depth="3" style="font-size: 12px">
        <code>{{ store.paths.localBin || 'shim 目录' }}</code>
      </NText>
      <NText depth="3" style="font-size: 12px">
        node {{ store.current.node?.version || '—' }} · java {{ store.current.java?.version || '—' }}
      </NText>
      <div style="flex: 1"></div>
      <NText depth="3" style="font-size: 12px">切换后当前终端立即生效</NText>
      <NText depth="3" style="font-size: 12px">v{{ store.version || '1.0.0' }}</NText>
    </NLayoutFooter>
  </NLayout>

  <NModal v-model:show="showAbout" preset="card" title="DevSwitch" style="width: 360px">
    <NText depth="3">v{{ store.version || '1.0.0' }}</NText>
  </NModal>
</template>
