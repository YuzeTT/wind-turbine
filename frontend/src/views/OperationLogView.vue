<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { listOperationLogs, createOperationLog } from '@/api/oplog'
import { listTurbines } from '@/api/turbine'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const logs = ref<Record<string, unknown>[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)

const filters = reactive({
  turbine_id: '',
  action: '',
})

const turbines = ref<{ id: number; name: string }[]>([])

const showCreateModal = ref(false)
const createForm = reactive({
  turbine_id: 0,
  operator: '',
  action: 'maintenance',
  reason: '',
})

const actionLabels: Record<string, string> = {
  fault_report: '故障上报',
  maintenance: '维修停机',
  weather_stop: '天气停机',
  manual_stop: '手动停机',
  manual_start: '手动启动',
  restart: '重启',
}

const actionTagTypes: Record<string, '' | 'success' | 'info' | 'warning' | 'danger' | 'primary'> = {
  fault_report: 'danger',
  maintenance: 'warning',
  weather_stop: 'warning',
  manual_stop: 'info',
  manual_start: 'success',
  restart: 'primary',
}

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

async function fetchLogs() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (filters.turbine_id) params.turbine_id = filters.turbine_id
    if (filters.action) params.action = filters.action

    const res = await listOperationLogs(params as never)
    logs.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

async function fetchTurbines() {
  const res = await listTurbines()
  turbines.value = res.data.map((t: { id: number; name: string }) => ({ id: t.id, name: t.name }))
}

function onFilter() {
  page.value = 1
  fetchLogs()
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    fetchLogs()
  }
}

function nextPage() {
  if (page.value < totalPages.value) {
    page.value++
    fetchLogs()
  }
}

function openCreateModal() {
  createForm.turbine_id = turbines.value[0]?.id || 0
  createForm.operator = auth.user?.nickname || ''
  createForm.action = 'maintenance'
  createForm.reason = ''
  showCreateModal.value = true
}

async function submitCreate() {
  try {
    await createOperationLog(createForm)
    showCreateModal.value = false
    await fetchLogs()
  } catch (err) {
    ElMessage.error((err as Error).message)
  }
}

onMounted(() => {
  fetchLogs()
  fetchTurbines()
})
</script>

<template>
  <div class="space-y-4">
    <!-- 筛选栏 -->
    <div class="flex flex-wrap items-center gap-3 mb-3!">
      <el-select v-model="filters.turbine_id" placeholder="全部风机" clearable style="width: 200px" @change="onFilter">
        <el-option label="全部风机" value="" />
        <el-option v-for="t in turbines" :key="t.id" :label="t.name" :value="t.id" />
      </el-select>

      <el-select v-model="filters.action" placeholder="全部操作" clearable style="width: 160px" @change="onFilter">
        <el-option label="全部操作" value="" />
        <el-option v-for="(label, key) in actionLabels" :key="key" :label="label" :value="key" />
      </el-select>

      <el-button v-if="auth.isAdmin" type="primary" class="ml-auto" @click="openCreateModal">
        + 新增操作
      </el-button>
    </div>

    <!-- 日志列表 -->
    <el-table :data="logs" v-loading="loading" border stripe style="width: 100%">
      <el-table-column label="风机" min-width="120">
        <template #default="{ row }">
          {{ (row.turbine as { name: string })?.name }}
        </template>
      </el-table-column>
      <el-table-column label="操作人" prop="operator" min-width="100" />
      <el-table-column label="操作类型" align="center" min-width="110">
        <template #default="{ row }">
          <el-tag :type="actionTagTypes[row.action as string] || 'info'" size="small">
            {{ actionLabels[row.action as string] || row.action }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="原因" prop="reason" min-width="180" />
      <el-table-column label="状态变更" align="center" min-width="140">
        <template #default="{ row }">
          <span class="text-xs text-gray-500">
            {{ row.prev_status }} → {{ row.new_status }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="时间" min-width="160">
        <template #default="{ row }">
          <span class="text-xs text-gray-500">
            {{ (row.created_at as string)?.substring(0, 19)?.replace('T', ' ') }}
          </span>
        </template>
      </el-table-column>

      <template #empty>
        <span class="text-gray-400">暂无操作日志</span>
      </template>
    </el-table>

    <!-- 分页 -->
    <div class="flex items-center justify-between px-4 py-3 border-t text-sm text-gray-500">
      <span>共 {{ total }} 条，第 {{ page }} / {{ totalPages }} 页</span>
      <div class="flex gap-2">
        <el-button size="small" :disabled="page <= 1" @click="prevPage">上一页</el-button>
        <el-button size="small" :disabled="page >= totalPages" @click="nextPage">下一页</el-button>
      </div>
    </div>

    <!-- 新增操作弹窗 -->
    <el-dialog v-model="showCreateModal" title="新增操作" width="420px" :close-on-click-modal="false">
      <el-form label-position="top" class="space-y-3">
        <el-form-item label="风机">
          <el-select v-model="createForm.turbine_id" placeholder="请选择风机" style="width: 100%">
            <el-option v-for="t in turbines" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>

        <div class="grid grid-cols-2 gap-3">
          <el-form-item label="操作人">
            <el-input v-model="createForm.operator" />
          </el-form-item>
          <el-form-item label="操作类型">
            <el-select v-model="createForm.action" style="width: 100%">
              <el-option v-for="(label, key) in actionLabels" :key="key" :label="label" :value="key" />
            </el-select>
          </el-form-item>
        </div>

        <el-form-item label="原因">
          <el-input v-model="createForm.reason" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="flex gap-2 justify-end">
          <el-button @click="showCreateModal = false">取消</el-button>
          <el-button type="primary" @click="submitCreate">确认</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>
