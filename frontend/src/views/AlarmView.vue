<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { listAlarms, createAlarm, resolveAlarm, alarmStats } from '@/api/alarm'
import { listTurbines } from '@/api/turbine'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const alarms = ref<Record<string, unknown>[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)

const filters = reactive({
  status: '',
  severity: '',
  turbine_id: '',
})

const statsData = ref<Record<string, unknown>>({})
const turbines = ref<{ id: number; name: string }[]>([])

// 创建报警弹窗
const showCreateModal = ref(false)
const createForm = reactive({
  turbine_id: 0,
  type: 'vibration',
  severity: 'warning',
  title: '',
  description: '',
  operator: '',
  stop_turbine: false,
})

// 处理报警弹窗
const showResolveModal = ref(false)
const resolveTarget = ref<Record<string, unknown>>({})
const resolveForm = reactive({
  resolved_by: '',
  comment: '',
  restart: false,
})

const severityLabels: Record<string, string> = {
  info: '信息',
  warning: '警告',
  critical: '严重',
}

const severityColors: Record<string, string> = {
  info: 'bg-blue-100 text-blue-600',
  warning: 'bg-yellow-100 text-yellow-600',
  critical: 'bg-red-100 text-red-600',
}

const alarmTypeLabels: Record<string, string> = {
  high_temp: '高温',
  vibration: '振动',
  pitch: '变桨',
  yaw: '偏航',
  grid: '电网',
  converter: '变流器',
  gearbox: '齿轮箱',
  wind_sensor: '风传感器',
}

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

async function fetchAlarms() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (filters.status) params.status = filters.status
    if (filters.severity) params.severity = filters.severity
    if (filters.turbine_id) params.turbine_id = filters.turbine_id

    const res = await listAlarms(params as never)
    alarms.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

async function fetchStats() {
  const res = await alarmStats()
  statsData.value = res.data
}

async function fetchTurbines() {
  const res = await listTurbines()
  turbines.value = res.data.map((t: { id: number; name: string }) => ({ id: t.id, name: t.name }))
}

function onFilter() {
  page.value = 1
  fetchAlarms()
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    fetchAlarms()
  }
}

function nextPage() {
  if (page.value < totalPages.value) {
    page.value++
    fetchAlarms()
  }
}

function openCreateModal() {
  createForm.turbine_id = turbines.value[0]?.id || 0
  createForm.type = 'vibration'
  createForm.severity = 'warning'
  createForm.title = ''
  createForm.description = ''
  createForm.operator = auth.user?.nickname || ''
  createForm.stop_turbine = false
  showCreateModal.value = true
}

async function submitCreate() {
  try {
    await createAlarm(createForm)
    showCreateModal.value = false
    await fetchAlarms()
    await fetchStats()
  } catch (err) {
    ElMessage.error((err as Error).message)
  }
}

function openResolveModal(alarm: Record<string, unknown>) {
  resolveTarget.value = alarm
  resolveForm.resolved_by = auth.user?.nickname || ''
  resolveForm.comment = ''
  resolveForm.restart = false
  showResolveModal.value = true
}

async function submitResolve() {
  try {
    await resolveAlarm(resolveTarget.value.id as number, resolveForm)
    showResolveModal.value = false
    await fetchAlarms()
    await fetchStats()
  } catch (err) {
    ElMessage.error((err as Error).message)
  }
}

onMounted(() => {
  fetchAlarms()
  fetchStats()
  fetchTurbines()
})

function getSeverityTagType(severity: string): 'danger' | 'warning' | 'primary' | 'info' {
  switch (severity) {
    case 'critical': return 'danger'
    case 'warning': return 'warning'
    case 'info': return 'info'
    default: return 'info'
  }
}

function getStatusTagType(status: string): 'warning' | 'success' {
  return status === 'active' ? 'warning' : 'success'
}

function handlePageChange(newPage: number) {
  page.value = newPage
  fetchAlarms()
}

function handlePageSizeChange(newSize: number) {
  pageSize.value = newSize
  page.value = 1
  fetchAlarms()
}
</script>

<template>
  <div class="space-y-4">
    <!-- 报警统计 -->
    <el-row :gutter="16">
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="活跃报警" :value="(statsData.active as number) ?? 0">
            <template #value>
              <span style="color: #e6a23c">{{ statsData.active ?? '-' }}</span>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="今日新增">
            <template #value>
              <span style="color: #409eff">{{ statsData.today ?? '-' }}</span>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="信息级">
            <template #value>
              <span style="color: #409eff">
                {{(statsData.by_severity as { severity: string; count: number }[])?.find(s => s.severity ===
                  'info')?.count
                  ?? 0}}
              </span>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="严重级">
            <template #value>
              <span style="color: #f56c6c">
                {{(statsData.by_severity as { severity: string; count: number }[])?.find(s => s.severity ===
                  'critical')?.count ?? 0}}
              </span>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
    </el-row>

    <!-- 筛选栏 -->
    <div class="flex flex-wrap items-center gap-3 my-3!">
      <el-select v-model="filters.status" placeholder="全部状态" clearable style="width: 140px" @change="onFilter">
        <el-option label="全部状态" value="" />
        <el-option label="活跃" value="active" />
        <el-option label="已处理" value="resolved" />
      </el-select>

      <el-select v-model="filters.severity" placeholder="全部级别" clearable style="width: 140px" @change="onFilter">
        <el-option label="全部级别" value="" />
        <el-option label="信息" value="info" />
        <el-option label="警告" value="warning" />
        <el-option label="严重" value="critical" />
      </el-select>

      <el-select v-model="filters.turbine_id" placeholder="全部风机" clearable style="width: 160px" @change="onFilter">
        <el-option label="全部风机" value="" />
        <el-option v-for="t in turbines" :key="t.id" :label="t.name" :value="t.id" />
      </el-select>

      <el-button v-if="auth.isAdmin" type="danger" class="ml-auto" @click="openCreateModal">
        + 上报故障
      </el-button>
    </div>

    <!-- 报警列表 -->
    <el-card shadow="never">
      <el-table :data="alarms" v-loading="loading" style="width: 100%" border>
        <el-table-column label="报警编号" prop="code" width="150">
          <template #default="{ row }">
            <span style="font-family: monospace; color: #909399">{{ row.code }}</span>
          </template>
        </el-table-column>
        <el-table-column label="风机" min-width="120">
          <template #default="{ row }">
            {{ (row.turbine as { name: string })?.name }}
          </template>
        </el-table-column>
        <el-table-column label="类型" width="100" align="center">
          <template #default="{ row }">
            {{ alarmTypeLabels[row.type as string] || row.type }}
          </template>
        </el-table-column>
        <el-table-column label="级别" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getSeverityTagType(row.severity as string)" size="small">
              {{ severityLabels[row.severity as string] || row.severity }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="标题" prop="title" min-width="160" />
        <el-table-column label="来源" width="80" align="center">
          <template #default="{ row }">
            <span style="font-size: 12px; color: #909399">
              {{ row.source === 'auto' ? '自动' : '手动' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status as string)" size="small">
              {{ row.status === 'active' ? '活跃' : '已处理' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="170">
          <template #default="{ row }">
            <span style="font-size: 12px; color: #909399">
              {{ (row.created_at as string)?.substring(0, 19)?.replace('T', ' ') }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" align="center" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status === 'active' && auth.isAdmin" type="success" size="small" text
              @click="openResolveModal(row)">
              处理
            </el-button>
          </template>
        </el-table-column>
        <template #empty>
          <span style="color: #909399">暂无报警记录</span>
        </template>
      </el-table>

      <!-- 分页 -->
      <div class="flex items-center justify-between mt-4">
        <span class="text-sm text-gray-500">
          共 {{ total }} 条，第 {{ page }} / {{ totalPages }} 页
        </span>
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total"
          :page-sizes="[10, 20, 50, 100]" layout="sizes, prev, pager, next, jumper" background
          @current-change="handlePageChange" @size-change="handlePageSizeChange" />
      </div>
    </el-card>

    <!-- 上报故障弹窗 -->
    <el-dialog v-model="showCreateModal" title="手动上报故障" width="500px">
      <el-form label-width="80px" :model="createForm">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="风机">
              <el-select v-model="createForm.turbine_id" placeholder="请选择风机" style="width: 100%">
                <el-option v-for="t in turbines" :key="t.id" :label="t.name" :value="t.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="操作人">
              <el-input v-model="createForm.operator" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="报警类型">
              <el-select v-model="createForm.type" placeholder="请选择类型" style="width: 100%">
                <el-option v-for="(label, key) in alarmTypeLabels" :key="key" :label="label" :value="key" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="级别">
              <el-select v-model="createForm.severity" placeholder="请选择级别" style="width: 100%">
                <el-option label="信息" value="info" />
                <el-option label="警告" value="warning" />
                <el-option label="严重" value="critical" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="标题">
          <el-input v-model="createForm.title" placeholder="故障描述" />
        </el-form-item>
        <el-form-item label="详细描述">
          <el-input v-model="createForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="createForm.stop_turbine">同时停机</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateModal = false">取消</el-button>
        <el-button type="danger" @click="submitCreate">上报</el-button>
      </template>
    </el-dialog>

    <!-- 处理报警弹窗 -->
    <el-dialog v-model="showResolveModal" title="处理报警" width="420px">
      <p class="text-sm text-gray-600 mb-4">{{ resolveTarget.title }}</p>
      <el-form label-width="80px" :model="resolveForm">
        <el-form-item label="处理人">
          <el-input v-model="resolveForm.resolved_by" />
        </el-form-item>
        <el-form-item label="处理备注">
          <el-input v-model="resolveForm.comment" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="resolveForm.restart">处理后恢复运行</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showResolveModal = false">取消</el-button>
        <el-button type="success" @click="submitResolve">确认处理</el-button>
      </template>
    </el-dialog>
  </div>
</template>
