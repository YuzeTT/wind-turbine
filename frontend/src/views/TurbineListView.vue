<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { listTurbines } from '@/api/turbine'
import { connectWebSocket, type TurbineData } from '@/api/websocket'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const turbines = ref<TurbineData[]>([])
const filterStatus = ref('')
const ws = ref<WebSocket | null>(null)

const statusColors: Record<string, string> = {
  running: 'bg-green-100 text-green-700 border-green-300',
  fault: 'bg-red-100 text-red-700 border-red-300',
  maintenance: 'bg-yellow-100 text-yellow-700 border-yellow-300',
  weather_stop: 'bg-orange-100 text-orange-700 border-orange-300',
  standby: 'bg-gray-100 text-gray-600 border-gray-300',
}

const statusLabels: Record<string, string> = {
  running: '运行中',
  fault: '故障',
  maintenance: '维护',
  weather_stop: '天气停机',
  standby: '待机',
}

const statusTagType: Record<string, 'success' | 'danger' | 'warning' | 'info' | 'primary'> = {
  running: 'success',
  fault: 'danger',
  maintenance: 'warning',
  weather_stop: 'warning',
  standby: 'info',
}

const statusFilters = [
  { label: '全部', value: '' },
  { label: '运行中', value: 'running' },
  { label: '故障', value: 'fault' },
  { label: '维护', value: 'maintenance' },
  { label: '天气停机', value: 'weather_stop' },
  { label: '待机', value: 'standby' },
]

const filteredTurbines = computed(() => {
  if (!filterStatus.value) return turbines.value
  return turbines.value.filter((t) => t.status === filterStatus.value)
})

const totalPower = computed(() => turbines.value.reduce((s, t) => s + t.power, 0))

async function fetchTurbines() {
  try {
    const res = await listTurbines()
    turbines.value = res.data
  } catch {
    // ignore
  }
}

function connectWS() {
  if (!auth.token) return
  ws.value = connectWebSocket(auth.token, {
    onTurbineUpdate: (turbine) => {
      // 增量更新：只更新这一台风机
      const idx = turbines.value.findIndex((t) => t.id === turbine.id)
      if (idx >= 0) {
        turbines.value[idx] = turbine
      } else {
        turbines.value.push(turbine)
      }
    },
  })
}

onMounted(() => {
  fetchTurbines()
  connectWS()
})

onUnmounted(() => {
  ws.value?.close()
})
</script>

<template>
  <div class="space-y-4">
    <!-- 筛选栏 -->
    <div class="flex items-center justify-between mb-3!">
      <el-radio-group v-model="filterStatus">
        <el-radio-button v-for="f in statusFilters" :key="f.value" :value="f.value">
          {{ f.label }}
        </el-radio-button>
      </el-radio-group>
      <div class="text-sm text-gray-500">
        共 {{ filteredTurbines.length }} 台 · 总功率 {{ totalPower.toFixed(0) }} kW
      </div>
    </div>

    <!-- 表格 -->
    <el-table :data="filteredTurbines" style="width: 100%" stripe border height="calc(100vh - 220px)"
      :header-cell-style="{ background: '#f9fafb', color: '#6b7280', fontWeight: 600 }">
      <el-table-column label="编号" min-width="140" fixed="left">
        <template #default="{ row }">
          <RouterLink :to="`/turbines/${row.id}`" class="text-blue-500 font-medium hover:underline">
            {{ row.name }}
          </RouterLink>
        </template>
      </el-table-column>

      <el-table-column prop="model" label="机型" min-width="100" sortable>
        <template #default="{ row }">
          <span class="text-gray-600 text-xs">{{ row.model }}</span>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="120" align="center" sortable
        :sort-method="(a: TurbineData, b: TurbineData) => a.status.localeCompare(b.status)">
        <template #default="{ row }">
          <el-tag :type="statusTagType[row.status]" size="small" effect="light">
            {{ statusLabels[row.status] || row.status }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="功率(kW)" width="110" align="right" sortable prop="power">
        <template #default="{ row }">
          <span :style="{ color: row.power > 0 ? '#16a34a' : '#9ca3af', fontFamily: 'monospace', fontWeight: 500 }">
            {{ row.power.toFixed(1) }}
          </span>
        </template>
      </el-table-column>

      <el-table-column prop="rated_power" label="额定(kW)" width="100" align="right" sortable>
        <template #default="{ row }">
          <span class="text-gray-500" style="font-family: monospace">{{ row.rated_power }}</span>
        </template>
      </el-table-column>

      <el-table-column label="转速(rpm)" width="110" align="right" sortable prop="rotor_speed">
        <template #default="{ row }">
          <span class="text-gray-600" style="font-family: monospace">{{ row.rotor_speed.toFixed(1) }}</span>
        </template>
      </el-table-column>

      <el-table-column label="风速(m/s)" width="110" align="right" sortable prop="wind_speed">
        <template #default="{ row }">
          <span class="text-gray-600" style="font-family: monospace">{{ row.wind_speed.toFixed(1) }}</span>
        </template>
      </el-table-column>

      <el-table-column label="机舱温度(℃)" width="120" align="right" sortable prop="temperature">
        <template #default="{ row }">
          <span :style="{ color: row.temperature > 50 ? '#ef4444' : '#4b5563', fontFamily: 'monospace' }">
            {{ row.temperature.toFixed(1) }}
          </span>
        </template>
      </el-table-column>

      <el-table-column label="今日发电(kWh)" width="130" align="right" sortable prop="today_power">
        <template #default="{ row }">
          <span class="text-gray-600" style="font-family: monospace">{{ row.today_power.toFixed(1) }}</span>
        </template>
      </el-table-column>

      <el-table-column label="可用率(%)" width="110" align="right" sortable prop="availability">
        <template #default="{ row }">
          <span :style="{ color: row.availability > 95 ? '#16a34a' : '#f97316', fontFamily: 'monospace' }">
            {{ row.availability.toFixed(1) }}
          </span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
