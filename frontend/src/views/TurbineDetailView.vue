<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getTurbine, updateTurbineStatus } from '@/api/turbine'
import { useAuthStore } from '@/stores/auth'
import WindTurbineIcon from '@/components/WindTurbineIcon.vue'
import { formatDateTime } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const turbine = ref<Record<string, unknown>>({})
const alarms = ref<Record<string, unknown>[]>([])
const stats = ref<Record<string, unknown>[]>([])
const loading = ref(true)

// 状态切换弹窗
const showStatusModal = ref(false)
const statusForm = ref({
  status: '',
  operator: '',
  reason: '',
})

const statusLabels: Record<string, string> = {
  running: '运行中',
  fault: '故障',
  maintenance: '维护',
  weather_stop: '天气停机',
  standby: '待机',
}

const statusTagTypes: Record<string, 'success' | 'danger' | 'warning' | 'info'> = {
  running: 'success',
  fault: 'danger',
  maintenance: 'warning',
  weather_stop: 'warning',
  standby: 'info',
}

const severityTagTypes: Record<string, 'danger' | 'warning' | 'primary'> = {
  critical: 'danger',
  warning: 'warning',
  info: 'primary',
}

const severityLabels: Record<string, string> = {
  critical: '严重',
  warning: '警告',
  info: '提示',
}

async function fetchData() {
  loading.value = true
  try {
    const res = await getTurbine(route.params.id as string)
    turbine.value = res.data.turbine
    alarms.value = res.data.alarms
    stats.value = res.data.stats
  } catch {
    // ignore
  } finally {
    loading.value = false
  }
}

function openStatusModal() {
  statusForm.value.status = ''
  statusForm.value.operator = auth.user?.nickname || ''
  statusForm.value.reason = ''
  showStatusModal.value = true
}

async function submitStatusChange() {
  try {
    await updateTurbineStatus(turbine.value.id as number, statusForm.value)
    showStatusModal.value = false
    await fetchData()
  } catch (err) {
    ElMessage.error((err as Error).message)
  }
}

onMounted(fetchData)
</script>

<template>
  <div class="turbine-detail">
    <el-page-header
      title="返回列表"
      :content="turbine.name as string || '风机详情'"
      @back="router.back()"
    >
      <template #extra>
        <el-button
          v-if="auth.isAdmin"
          type="primary"
          @click="openStatusModal"
        >
          切换状态
        </el-button>
      </template>
    </el-page-header>

    <!-- 风机可视化头部 -->
    <el-card v-if="!loading" shadow="hover" class="turbine-header-card">
      <div class="turbine-header-inner">
        <WindTurbineIcon
          :status="turbine.status as string"
          :rotor-speed="turbine.rotor_speed as number"
          :size="80"
        />
        <div class="turbine-header-info">
          <h2 class="turbine-title">{{ turbine.name }}</h2>
          <p class="turbine-subtitle">{{ turbine.model }} · {{ turbine.farm }}</p>
          <el-tag
            :type="statusTagTypes[turbine.status as string] || 'info'"
            effect="dark"
            size="large"
          >
            {{ statusLabels[turbine.status as string] || turbine.status }}
          </el-tag>
        </div>
      </div>
    </el-card>

    <div v-loading="loading" class="content-area">
      <template v-if="!loading">
        <!-- 基本信息 -->
        <el-descriptions
          :title="turbine.name as string"
          :column="3"
          border
          class="info-descriptions"
        >
          <el-descriptions-item label="型号">
            {{ turbine.model }}
          </el-descriptions-item>
          <el-descriptions-item label="所属风场">
            {{ turbine.farm }}
          </el-descriptions-item>
          <el-descriptions-item label="额定功率">
            {{ turbine.rated_power }} kW
          </el-descriptions-item>
          <el-descriptions-item label="投运日期">
            {{ (turbine.installed_date as string)?.substring(0, 10) }}
          </el-descriptions-item>
          <el-descriptions-item label="当前状态">
            <el-tag
              :type="statusTagTypes[turbine.status as string] || 'info'"
              effect="light"
            >
              {{ statusLabels[turbine.status as string] || turbine.status }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="可用率">
            {{ (turbine.availability as number)?.toFixed(1) }}%
          </el-descriptions-item>
        </el-descriptions>

        <!-- 实时参数 -->
        <el-row :gutter="16" class="section">
          <el-col :xs="24" :sm="12" :md="8" :lg="4">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">当前功率</p>
              <p class="param-value text-blue">{{ (turbine.power as number)?.toFixed(1) || '0' }}</p>
              <p class="param-unit">kW</p>
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="4">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">转速</p>
              <p class="param-value text-cyan">{{ (turbine.rotor_speed as number)?.toFixed(1) || '0' }}</p>
              <p class="param-unit">rpm</p>
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="4">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">风速</p>
              <p class="param-value text-teal">{{ (turbine.wind_speed as number)?.toFixed(1) || '0' }}</p>
              <p class="param-unit">m/s</p>
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="4">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">风向</p>
              <p class="param-value text-indigo">{{ (turbine.wind_direction as number)?.toFixed(0) || '0' }}</p>
              <p class="param-unit">°</p>
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="4">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">机舱温度</p>
              <p
                class="param-value"
                :class="(turbine.temperature as number) > 50 ? 'text-red' : 'text-green'"
              >
                {{ (turbine.temperature as number)?.toFixed(1) || '0' }}
              </p>
              <p class="param-unit">℃</p>
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="4">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">可用率</p>
              <p class="param-value text-purple">{{ (turbine.availability as number)?.toFixed(1) || '0' }}</p>
              <p class="param-unit">%</p>
            </el-card>
          </el-col>
        </el-row>

        <!-- 累计信息 -->
        <el-row :gutter="16" class="section">
          <el-col :xs="24" :sm="8">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">今日发电量</p>
              <p class="param-value text-cyan">{{ (turbine.today_power as number)?.toFixed(1) }} kWh</p>
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="8">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">累计发电量</p>
              <p class="param-value text-blue">
                {{ ((turbine.total_power as number) / 10000)?.toFixed(1) }} 万 kWh
              </p>
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="8">
            <el-card shadow="hover" body-style="text-align:center;padding:16px;">
              <p class="param-label">投运日期</p>
              <p class="param-value text-gray">
                {{ (turbine.installed_date as string)?.substring(0, 10) }}
              </p>
            </el-card>
          </el-col>
        </el-row>

        <el-row :gutter="16" class="section">
          <!-- 近期报警 -->
          <el-col :xs="24" :lg="12">
            <el-card shadow="never" class="section-card">
              <template #header>
                <span class="card-title">近期报警</span>
              </template>
              <el-timeline v-if="alarms.length > 0">
                <el-timeline-item
                  v-for="a in alarms"
                  :key="a.id as number"
                  :timestamp="formatDateTime(a.created_at as string)"
                  placement="top"
                  :type="severityTagTypes[a.severity as string] || 'primary'"
                >
                  <div class="alarm-item">
                    <div class="alarm-header">
                      <span class="alarm-title">{{ a.title }}</span>
                      <el-tag
                        size="small"
                        :type="severityTagTypes[a.severity as string] || 'primary'"
                        effect="light"
                      >
                        {{ severityLabels[a.severity as string] || a.severity }}
                      </el-tag>
                    </div>
                    <p class="alarm-desc">{{ a.description }}</p>
                  </div>
                </el-timeline-item>
              </el-timeline>
              <el-empty v-else description="暂无报警记录" :image-size="80" />
            </el-card>
          </el-col>

          <!-- 近 7 天统计 -->
          <el-col :xs="24" :lg="12">
            <el-card shadow="never" class="section-card">
              <template #header>
                <span class="card-title">近 7 天统计</span>
              </template>
              <el-table
                v-if="stats.length > 0"
                :data="stats"
                size="small"
                stripe
                style="width:100%"
              >
                <el-table-column prop="date" label="日期" min-width="110" />
                <el-table-column label="发电量" min-width="100">
                  <template #default="{ row }">
                    {{ (row.total_power as number)?.toFixed(0) }} kWh
                  </template>
                </el-table-column>
                <el-table-column label="最大功率" min-width="90">
                  <template #default="{ row }">
                    {{ (row.max_power as number)?.toFixed(0) }} kW
                  </template>
                </el-table-column>
                <el-table-column label="平均功率" min-width="90">
                  <template #default="{ row }">
                    {{ (row.avg_power as number)?.toFixed(0) }} kW
                  </template>
                </el-table-column>
                <el-table-column label="可用率" min-width="80">
                  <template #default="{ row }">
                    {{ (row.availability as number)?.toFixed(1) }}%
                  </template>
                </el-table-column>
                <el-table-column label="运行小时" min-width="90">
                  <template #default="{ row }">
                    {{ (row.run_hours as number)?.toFixed(1) }}h
                  </template>
                </el-table-column>
              </el-table>
              <el-empty v-else description="暂无统计数据" :image-size="80" />
            </el-card>
          </el-col>
        </el-row>
      </template>
    </div>

    <!-- 状态切换弹窗 -->
    <el-dialog
      v-model="showStatusModal"
      title="切换风机状态"
      width="420px"
      :close-on-click-modal="false"
    >
      <el-form :model="statusForm" label-width="80px" label-position="right">
        <el-form-item label="目标状态">
          <el-select v-model="statusForm.status" placeholder="请选择" style="width:100%">
            <el-option label="运行" value="running" />
            <el-option label="待机" value="standby" />
            <el-option label="维护" value="maintenance" />
            <el-option label="天气停机" value="weather_stop" />
          </el-select>
        </el-form-item>
        <el-form-item label="操作人">
          <el-input v-model="statusForm.operator" />
        </el-form-item>
        <el-form-item label="操作原因">
          <el-input
            v-model="statusForm.reason"
            type="textarea"
            :rows="2"
            placeholder="请输入操作原因"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showStatusModal = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="!statusForm.status"
          @click="submitStatusChange"
        >
          确认
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.turbine-detail {
  padding: 16px;
}

.content-area {
  margin-top: 16px;
}

.info-descriptions {
  margin-bottom: 16px;
}

.section {
  margin-bottom: 16px;
}

.section-card {
  height: 100%;
}

.card-title {
  font-weight: 600;
  font-size: 15px;
}

.param-label {
  font-size: 12px;
  color: #909399;
  margin: 0 0 4px;
}

.param-value {
  font-size: 22px;
  font-weight: 700;
  margin: 0 0 2px;
}

.param-unit {
  font-size: 12px;
  color: #c0c4cc;
  margin: 0;
}

.alarm-item {
  line-height: 1.5;
}

.alarm-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.alarm-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.alarm-desc {
  font-size: 12px;
  color: #909399;
  margin: 4px 0 0;
}

.text-blue { color: #409eff; }
.text-cyan { color: #13c2c2; }
.text-teal { color: #0fbf86; }
.text-indigo { color: #5b6ef5; }
.text-red { color: #f56c6c; }
.text-green { color: #67c23a; }
.text-purple { color: #a855f7; }
.text-gray { color: #606266; }

.turbine-header-card {
  margin-top: 16px;
  border-radius: 8px;
}

.turbine-header-inner {
  display: flex;
  align-items: center;
  gap: 24px;
}

.turbine-header-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.turbine-title {
  font-size: 24px;
  font-weight: 700;
  color: #303133;
  margin: 0;
}

.turbine-subtitle {
  font-size: 13px;
  color: #909399;
  margin: 0;
}
</style>
