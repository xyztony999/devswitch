<script setup lang="ts">
defineProps<{ kind: 'node' | 'java'; large?: boolean }>()
</script>

<template>
  <span :class="large ? 'card-logo' : 'brand-logo'" aria-hidden="true">
    <svg v-if="kind === 'node'" viewBox="0 0 32 32">
      <path fill="#5FA04E" d="M16 2.1 28.7 9.4v13.2L16 29.9 3.3 22.6V9.4L16 2.1z" />
      <path fill="#3F7E33" d="M16 2.1v27.8l12.7-7.3V9.4L16 2.1z" />
      <path fill="#8CC84B" d="M16 6.4 24 11v10l-8 4.6L8 21V11l8-4.6z" />
      <path
        fill="#fff"
        d="M14.3 13.2c0-1.3.9-2.1 2.4-2.1 1.1 0 1.9.4 2.4 1.1l-1.2.8c-.3-.4-.6-.6-1.2-.6-.5 0-.8.3-.8.7 0 .4.3.6 1 .8l.8.2c1.4.4 2.1 1.1 2.1 2.3 0 1.4-1.1 2.3-2.7 2.3-1.3 0-2.3-.5-2.8-1.4l1.3-.8c.3.6.8.9 1.5.9.6 0 1-.3 1-.8 0-.4-.3-.6-1-.8l-.8-.2c-1.4-.4-2-1.1-2-2.4z"
      />
    </svg>
    <svg v-else viewBox="0 0 32 32">
      <path
        fill="none"
        stroke="#E76F00"
        stroke-width="1.6"
        stroke-linecap="round"
        d="M12 4.5c0 1.8 2 1.8 2 3.6s-2 1.8-2 3.6"
      />
      <path
        fill="none"
        stroke="#E76F00"
        stroke-width="1.6"
        stroke-linecap="round"
        d="M16 3.5c0 1.8 2 1.8 2 3.6s-2 1.8-2 3.6"
      />
      <path fill="#E76F00" d="M8.5 13h13v7.2a5.8 5.8 0 0 1-5.8 5.8h-1.4A5.8 5.8 0 0 1 8.5 20.2V13z" />
      <path
        fill="none"
        stroke="#E76F00"
        stroke-width="1.8"
        stroke-linecap="round"
        d="M21.5 15.2h2.2a2.6 2.6 0 1 1 0 5.2h-2.2"
      />
      <rect x="8" y="27" width="14.5" height="1.8" rx=".9" fill="#E76F00" />
    </svg>
  </span>
</template>
