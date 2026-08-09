<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { overview, statusDistribution, powerTrend, dailyEnergy } from '@/api/dashboard'
import { connectWebSocket, type TurbineData } from '@/api/websocket'
import { updateTurbineStatus } from '@/api/turbine'
import { listTurbines } from '@/api/turbine'
import WindTurbineIcon from '@/components/WindTurbineIcon.vue'

const auth = useAuthStore()
const router = useRouter()

const overviewData = ref<Record<string, unknown>>({})
const statusData = ref<{ status: string; count: number }[]>([])
const trendData = ref<{ hour: string; total_power: number; avg_power: number }[]>([])
const energyData = ref<{ date: string; total_energy: number }[]>([])

// baseTurbines: WS 推送的原始值（按 id 索引）
// displayTurbines: 抖动后用于展示的数组
const baseTurbines = ref<TurbineData[]>([])
const displayTurbines = ref<TurbineData[]>([])
const wsUpdateTime = ref('')
let ws: WebSocket | null = null

// 连接状态
const isConnected = ref(false)
let lastHeartbeat = 0
let heartbeatWatcher: ReturnType<typeof setInterval> | null = null

// 数字闪烁
const flashEnabled = ref(true)
const flashingIds = ref<Set<number>>(new Set())

const statusLabels: Record<string, string> = {
  running: '正常', fault: '故障', maintenance: '维护', weather_stop: '停机', standby: '待机',
}
const statusTagType: Record<string, 'success' | 'danger' | 'warning' | 'info'> = {
  running: 'success', fault: 'danger', maintenance: 'warning', weather_stop: 'warning', standby: 'info',
}

async function fetchData() {
  try {
    // 先用 REST 拉取风机列表作为初始数据
    const tr = await listTurbines()
    baseTurbines.value = tr.data as TurbineData[]
    displayTurbines.value = baseTurbines.value.map((t) => ({ ...t }))

    const [o, s, t, e] = await Promise.all([overview(), statusDistribution(), powerTrend(), dailyEnergy()])
    overviewData.value = o.data
    statusData.value = s.data
    trendData.value = t.data
    energyData.value = e.data
  } catch { /* ignore */ }
}

function connectWS() {
  if (!auth.token) return
  ws = connectWebSocket(auth.token, {
    onOpen: () => { isConnected.value = true },
    onClose: () => { isConnected.value = false },
    onHeartbeat: () => { lastHeartbeat = Date.now() },
    onTurbineUpdate: (turbine, time) => {
      const idx = baseTurbines.value.findIndex((t) => t.id === turbine.id)
      if (idx >= 0) {
        if (baseTurbines.value[idx]?.status !== turbine.status) {
          delete pendingActions.value[turbine.id]
        }
        baseTurbines.value[idx] = turbine
        displayTurbines.value[idx] = { ...turbine }
      } else {
        baseTurbines.value.push(turbine)
        displayTurbines.value.push({ ...turbine })
      }
      wsUpdateTime.value = time

      if (flashEnabled.value && turbine.status === 'running') {
        const id = turbine.id
        flashingIds.value = new Set([id])
        setTimeout(() => { flashingIds.value = new Set() }, 600)
      }
    },
  })
}

// 按钮操作中状态
const pendingActions = ref<Record<number, string>>({})

function powerPct(t: TurbineData): number {
  return Math.min(100, Math.max(0, (t.power / t.rated_power) * 100))
}

onMounted(() => {
  fetchData()
  connectWS()

  heartbeatWatcher = setInterval(() => {
    if (lastHeartbeat > 0 && Date.now() - lastHeartbeat > 15000) {
      isConnected.value = false
    }
  }, 3000)
})

onUnmounted(() => {
  if (heartbeatWatcher) clearInterval(heartbeatWatcher)
  ws?.close()
})

const liveTotalPower = computed(() => displayTurbines.value.reduce((sum, t) => sum + t.power, 0))
// const liveRunningCount = computed(() => displayTurbines.value.filter((t) => t.status === 'running').length)

async function handleStop(id: number, name: string) {
  pendingActions.value[id] = 'stopping'
  try {
    await updateTurbineStatus(id, { status: 'maintenance', operator: auth.user?.nickname || '管理员', reason: '远程急停' })
    ElMessage.success(`${name} 已发送急停指令`)
  } catch (err) {
    ElMessage.error((err as Error).message)
    delete pendingActions.value[id]
  }
}

async function handleStart(id: number, name: string) {
  pendingActions.value[id] = 'starting'
  try {
    await updateTurbineStatus(id, { status: 'running', operator: auth.user?.nickname || '管理员', reason: '远程启动' })
    ElMessage.success(`${name} 已发送启动指令`)
  } catch (err) {
    ElMessage.error((err as Error).message)
    delete pendingActions.value[id]
  }
}
</script>

<template>
  <div class="dashboard-container">
    <!-- 指标卡片 -->
    <el-row :gutter="16" class="metric-row">
      <el-col :xs="12" :sm="8" :md="6" :lg="4">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-label">
            <el-icon>
              <DataLine />
            </el-icon>
            <span>风机总数</span>
          </div>
          <div class="metric-value">{{ overviewData.total_turbines ?? '-' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="6" :lg="4">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-label">
            <el-icon>
              <Promotion />
            </el-icon>
            <span>运行中</span>
          </div>
          <div class="metric-value text-success">
            {{ overviewData.running ?? '-' }}
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="6" :lg="4">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-label">
            <el-icon>
              <WarningFilled />
            </el-icon>
            <span>故障</span>
          </div>
          <div class="metric-value text-danger">
            {{ overviewData.fault ?? '-' }}
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="6" :lg="4">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-label">
            <el-icon>
              <Lightning />
            </el-icon>
            <span>当前总功率</span>
          </div>
          <div class="metric-value text-primary">
            {{ (liveTotalPower || (overviewData.total_power as number) || 0).toFixed(0) }}
            <span class="metric-unit">kW</span>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="6" :lg="4">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-label">
            <el-icon>
              <Sunrise />
            </el-icon>
            <span>今日发电量</span>
          </div>
          <div class="metric-value text-cyan">
            {{ ((overviewData.today_energy as number) ?? 0).toFixed(0) }}
            <span class="metric-unit">kWh</span>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="8" :md="6" :lg="4">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-label">
            <el-icon>
              <BellFilled />
            </el-icon>
            <span>活跃报警</span>
          </div>
          <div class="metric-value text-warning">
            {{ overviewData.active_alarms ?? '-' }}
          </div>
        </el-card>
      </el-col>
    </el-row>





    <!-- 实时风机状态 -->
    <el-row :gutter="16" class="panel-row">
      <el-col :span="24">
        <el-card shadow="hover" class="panel-card">
          <template #header>
            <div class="panel-header">
              <el-icon>
                <Monitor />
              </el-icon>
              <span>实时风机状态</span>
              <el-switch v-model="flashEnabled" active-text="闪烁" size="small" class="ml-2" />
              <span v-if="wsUpdateTime" class="ws-time">
                <el-icon>
                  <Clock />
                </el-icon>
                更新: {{ wsUpdateTime }}
              </span>
            </div>
          </template>
          <el-empty v-if="displayTurbines.length === 0" description="等待 WebSocket 连接..." :image-size="60" />
          <div v-else
            class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3 2xl:grid-cols-4 gap-2 max-h-[600px] overflow-y-auto p-1 ">
            <div v-for="t in displayTurbines" :key="t.id"
              class="flex  gap-2  border border-gray-200 hover:border-blue-400 hover:shadow-md transition-all bg-white">
              <!-- 竖向发电进度条 -->
              <div class="w-1.5 h-full bg-gray-200 overflow-hidden relative shrink-0 ">
                <div class="absolute bottom-0 left-0 right-0 transition-all duration-700 ease-out "
                  :class="powerPct(t) > 60 ? 'bg-green-500' : powerPct(t) > 25 ? 'bg-amber-500' : powerPct(t) > 0 ? 'bg-red-400' : 'bg-gray-300'"
                  :style="{ height: Math.max(2, powerPct(t)) + '%' }"></div>
              </div>

              <!-- 风机动画 -->
              <div>
                <RouterLink :to="`/turbines/${t.id}`" class="shrink-0 flex items-center justify-center w-10 h-16">
                  <WindTurbineIcon :status="t.status" :rotor-speed="t.rotor_speed" :size="34" />

                </RouterLink>
                <el-tag :type="statusTagType[t.status] || 'info'" effect="dark" size="small" round class="mb-2!">
                  {{ statusLabels[t.status] || t.status }}
                </el-tag>
              </div>

              <!-- 名称 + 状态 + 数据 -->
              <div class="flex-1 min-w-0 ml-2! mt-2!">
                <div class="flex items-center gap-1.5 mb-1!">
                  <RouterLink :to="`/turbines/${t.id}`">
                    <div class="text-sm font-semibold text-gray-800 hover:text-blue-500 truncate ">{{ t.name }}</div>
                    <div class="text-[10px] opacity-50">{{ t.farm }} · {{ t.model }}</div>
                  </RouterLink>
                  <!-- <el-tag :type="statusTagType[t.status] || 'info'" effect="dark" size="small" round>
                    {{ statusLabels[t.status] || t.status }}
                  </el-tag> -->
                </div>
                <div class="grid grid-cols-4 gap-1">
                  <div v-for="item in [
                    { label: '功率', value: t.power.toFixed(0), unit: 'kW', color: t.power > 0 ? 'text-green-600' : 'text-gray-300' },
                    { label: '风速', value: t.wind_speed.toFixed(1), unit: 'm/s', color: 'text-gray-700' },
                    { label: '转速', value: t.rotor_speed.toFixed(1), unit: 'rpm', color: 'text-gray-700' },
                    { label: '温度', value: t.temperature.toFixed(0), unit: '℃', color: t.temperature > 50 ? 'text-red-500' : 'text-gray-700' },
                  ]" :key="item.label" class="flex flex-col rounded px-0.5 transition-colors duration-300"
                    :class="flashEnabled && flashingIds.has(t.id) ? 'bg-blue-50' : ''">
                    <span class="text-[11px] text-gray-400">{{ item.label }}</span>
                    <span class="text-sm font-bold font-mono" :class="[
                      item.color,
                      flashEnabled && flashingIds.has(t.id) ? 'text-blue-600' : '',
                    ]">
                      {{ item.value }}<span class="text-[10px] font-normal text-gray-400 ml-0.5!">{{ item.unit }}</span>
                    </span>
                  </div>

                </div>
              </div>
              <!-- 操作按钮 -->
              <div class="mt-2! mr-2!">
                <!-- 操作中：显示 loading -->
                <el-button v-if="pendingActions[t.id]" size="small" type="info" loading disabled>
                  {{ pendingActions[t.id] === 'stopping' ? '急停中' : '启动中' }}
                </el-button>
                <!-- 运行中 → 急停 -->
                <el-button v-else-if="t.status === 'running'" type="danger" size="small" :disabled="!auth.isAdmin"
                  @click.prevent="handleStop(t.id, t.name)">急停</el-button>
                <!-- 故障 → 排障 -->
                <el-button v-else-if="t.status === 'fault'" type="warning" size="small"
                  @click.prevent="router.push({ name: 'alarms', query: { turbine_id: t.id } })">排障</el-button>
                <!-- 非运行非故障 → 启动 -->
                <el-button v-else type="success" size="small" :disabled="!auth.isAdmin"
                  @click.prevent="handleStart(t.id, t.name)">启动</el-button>
              </div>


            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 每日发电量 -->
    <el-row :gutter="16" class="panel-row">
      <el-col :span="24">
        <el-card shadow="hover" class="panel-card">
          <template #header>
            <div class="panel-header">
              <el-icon>
                <DataAnalysis />
              </el-icon>
              <span>近 7 日发电量</span>
            </div>
          </template>
          <div class="energy-list">
            <div v-for="e in energyData" :key="e.date" class="energy-item">
              <span class="energy-date">{{ e.date }}</span>
              <el-progress
                :percentage="Number(((e.total_energy / Math.max(...energyData.map(d => d.total_energy), 1)) * 100).toFixed(1))"
                :stroke-width="12" color="#13c2c2" :show-text="false" class="energy-progress" />
              <span class="energy-value">
                {{ (e.total_energy / 1000).toFixed(1) }} MWh
              </span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row">
      <!-- 状态分布 -->
      <el-col :xs="24" :lg="8">
        <el-card shadow="hover" class="panel-card">
          <template #header>
            <div class="panel-header">
              <el-icon>
                <PieChart />
              </el-icon>
              <span>状态分布</span>
            </div>
          </template>
          <div class="status-list">
            <div v-for="s in statusData" :key="s.status" class="status-item">
              <el-tag
                :type="({ running: 'success', fault: 'danger', maintenance: 'warning', weather_stop: 'warning', standby: 'info' } as Record<string, 'success' | 'danger' | 'warning' | 'info'>)[s.status] || 'info'"
                effect="dark" size="small">
                {{ statusLabels[s.status] || s.status }}
              </el-tag>
              <el-progress
                :percentage="Number(((s.count / ((overviewData.total_turbines as number) || 1)) * 100).toFixed(1))"
                :stroke-width="14"
                :color="({ running: '#67c23a', fault: '#f56c6c', maintenance: '#e6a23c', weather_stop: '#e6a23c', standby: '#909399' } as Record<string, string>)[s.status] || '#909399'"
                :show-text="false" class="status-progress" />
              <span class="status-count">{{ s.count }}</span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 功率趋势 -->
      <el-col :xs="24" :lg="16">
        <el-card shadow="hover" class="panel-card">
          <template #header>
            <div class="panel-header">
              <el-icon>
                <TrendCharts />
              </el-icon>
              <span>24h 功率趋势</span>
            </div>
          </template>
          <div class="trend-chart">
            <div v-for="(t, i) in trendData" :key="i" class="trend-bar-wrapper">
              <el-tooltip placement="top" :content="t.total_power.toFixed(0) + ' kW'">
                <div class="trend-bar"
                  :style="{ height: Math.max(4, (t.total_power / Math.max(...trendData.map(d => d.total_power), 1)) * 160) + 'px' }">
                </div>
              </el-tooltip>
              <span class="trend-hour">{{ t.hour }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 断连遮罩 -->
    <div v-if="!isConnected && displayTurbines.length > 0"
      class="absolute inset-0 z-50 bg-gray-900/70 backdrop-blur-sm flex items-center justify-center rounded-lg">
      <div class="text-center text-white">
        <el-icon size="48" class="animate-pulse mb-4">
          <WarningFilled />
        </el-icon>
        <p class="text-xl font-bold mb-2">数据源断连</p>
        <p class="text-sm text-gray-300">WebSocket 连接已断开，正在尝试重连…</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
  position: relative;
}

.metric-row {
  margin-bottom: 0 !important;
}

.metric-card {
  height: 100%;
  border-radius: 8px;
  transition: transform 0.2s ease;
}

.metric-card:hover {
  transform: translateY(-2px);
}

.metric-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}

.metric-label .el-icon {
  font-size: 16px;
  color: #c0c4cc;
}

.metric-value {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
  line-height: 1.2;
}

.metric-unit {
  font-size: 14px;
  font-weight: 400;
  color: #909399;
  margin-left: 2px;
}

.text-success {
  color: #67c23a;
}

.text-danger {
  color: #f56c6c;
}

.text-primary {
  color: #409eff;
}

.text-warning {
  color: #e6a23c;
}

.text-cyan {
  color: #13c2c2;
}

.panel-row {
  margin-bottom: 0 !important;
}

.panel-card {
  border-radius: 8px;
  height: 100%;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #303133;
}

.panel-header .el-icon {
  font-size: 18px;
  color: #409eff;
}

.ws-time {
  margin-left: auto;
  font-size: 12px;
  font-weight: 400;
  color: #c0c4cc;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

/* 状态分布 */
.status-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-item .el-tag {
  width: 80px;
  text-align: center;
}

.status-progress {
  flex: 1;
}

.status-count {
  width: 32px;
  text-align: right;
  font-size: 14px;
  font-weight: 600;
  color: #606266;
}

/* 功率趋势 */
.trend-chart {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 200px;
  padding: 8px 0;
}

.trend-bar-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  height: 100%;
  justify-content: flex-end;
}

.trend-bar {
  width: 100%;
  background: linear-gradient(to top, #409eff, #79bbff);
  border-radius: 4px 4px 0 0;
  transition: background 0.2s ease;
  cursor: pointer;
  min-height: 4px;
}

.trend-bar:hover {
  background: linear-gradient(to top, #337ecc, #409eff);
}

.trend-hour {
  font-size: 10px;
  color: #c0c4cc;
}

/* 每日发电量 */
.energy-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.energy-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.energy-date {
  width: 96px;
  font-size: 12px;
  color: #909399;
}

.energy-progress {
  flex: 1;
}

.energy-value {
  width: 96px;
  text-align: right;
  font-size: 12px;
  font-weight: 600;
  color: #606266;
}
</style>
