<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
  rotorSpeed?: number
  size?: number
}>()

const isRunning = computed(() => props.status === 'running' && (props.rotorSpeed || 0) > 0)

// 旋转周期：60 / rpm 秒，限制在 2~15 秒
const rotationDuration = computed(() => {
  const rpm = props.rotorSpeed || 0
  if (rpm <= 0) return '0s'
  return `${Math.max(2, Math.min(60 / rpm, 15))}s`
})

const colors = computed(() => {
  const fallback = { tower: '#94a3b8', blade: '#9ca3af', nacelle: '#64748b', hub: '#6b7280', glow: '#9ca3af' }
  const schemes: Record<string, typeof fallback> = {
    running:     { tower: '#94a3b8', blade: '#22c55e', nacelle: '#64748b', hub: '#16a34a', glow: '#22c55e' },
    fault:       { tower: '#94a3b8', blade: '#ef4444', nacelle: '#64748b', hub: '#dc2626', glow: '#ef4444' },
    maintenance: { tower: '#94a3b8', blade: '#f59e0b', nacelle: '#64748b', hub: '#d97706', glow: '#f59e0b' },
    weather_stop:{ tower: '#94a3b8', blade: '#f97316', nacelle: '#64748b', hub: '#ea580c', glow: '#f97316' },
    standby:     { tower: '#94a3b8', blade: '#9ca3af', nacelle: '#64748b', hub: '#6b7280', glow: '#9ca3af' },
  }
  return schemes[props.status] ?? fallback
})
</script>

<template>
  <div :style="{ width: (size || 80) + 'px', height: (size || 80) * 1.4 + 'px' }" class="relative">
    <svg viewBox="0 0 100 140" class="w-full h-full overflow-visible">
      <!-- 光晕（运行中） -->
      <circle v-if="isRunning" cx="50" cy="42" r="30" :fill="colors.glow" opacity="0.12" />

      <!-- 塔基 -->
      <rect x="43" y="128" width="14" height="10" rx="1" :fill="colors.tower" />
      <!-- 塔筒（锥形） -->
      <polygon points="46,128 54,128 52,44 48,44" :fill="colors.tower" />
      <!-- 机舱 -->
      <ellipse cx="50" cy="42" rx="8" ry="4.5" :fill="colors.nacelle" />

      <!-- 叶片组 -->
      <g
        class="blades"
        :class="{ spinning: isRunning }"
        style="transform-origin: 50px 42px; transform-box: view-box;"
        :style="{ animationDuration: rotationDuration }"
      >
        <!-- 三个叶片 0° 120° 240° -->
        <path
          d="M 50 42 Q 47 32 46.5 16 Q 47 5 50 3 Q 53 5 53.5 16 Q 53 32 50 42 Z"
          :fill="colors.blade"
          transform="rotate(0 50 42)"
        />
        <path
          d="M 50 42 Q 47 32 46.5 16 Q 47 5 50 3 Q 53 5 53.5 16 Q 53 32 50 42 Z"
          :fill="colors.blade"
          transform="rotate(120 50 42)"
        />
        <path
          d="M 50 42 Q 47 32 46.5 16 Q 47 5 50 3 Q 53 5 53.5 16 Q 53 32 50 42 Z"
          :fill="colors.blade"
          transform="rotate(240 50 42)"
        />
        <!-- 轮毂 -->
        <circle cx="50" cy="42" r="3.5" :fill="colors.hub" />
      </g>
    </svg>
  </div>
</template>

<style scoped>
.blades.spinning {
  animation: wind-blade-spin linear infinite;
}

@keyframes wind-blade-spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}
</style>
