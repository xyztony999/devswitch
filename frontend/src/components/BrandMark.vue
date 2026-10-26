<script setup lang="ts">
// DevSwitch 品牌标识（拨动开关），与 exe/托盘/安装器同源。
defineProps<{ size?: number }>()
</script>

<template>
  <svg
    class="brand-mark"
    :width="size || 26"
    :height="size || 26"
    viewBox="0 0 256 256"
    aria-hidden="true"
  >
    <defs>
      <linearGradient id="bm-bg" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#262a33" />
        <stop offset="1" stop-color="#15181e" />
      </linearGradient>
      <linearGradient id="bm-track" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#1fb564" />
        <stop offset="1" stop-color="#0f8a4c" />
      </linearGradient>
    </defs>
    <rect x="8" y="8" width="240" height="240" rx="56" fill="url(#bm-bg)" />
    <rect x="52" y="104" width="152" height="48" rx="24" fill="url(#bm-track)" />
    <circle cx="180" cy="128" r="17" fill="#f2f5f3" />
  </svg>
</template>

<style>
.brand-mark {
  display: inline-block;
  flex: 0 0 auto;
  border-radius: 22%;
}
</style>
