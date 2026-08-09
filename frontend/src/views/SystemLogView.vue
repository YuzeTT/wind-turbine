<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { listSystemLogs } from '@/api/syslog'

const logs = ref<Record<string, unknown>[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)

const filters = reactive({
  level: '',
  module: '',
})

const levelTagType: Record<string, '' | 'success' | 'info' | 'warning' | 'danger'> = {
  info: 'info',
  warning: 'warning',
  error: 'danger',
}

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

async function fetchLogs() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (filters.level) params.level = filters.level
    if (filters.module) params.module = filters.module

    const res = await listSystemLogs(params as never)
    logs.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
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

function onPageChange(target: number) {
  if (target > page.value) {
    nextPage()
  } else if (target < page.value) {
    prevPage()
  }
}

onMounted(fetchLogs)
</script>

<template>
  <div class="space-y-4">
    <!-- 筛选栏 -->
    <div class="flex items-center gap-3 mb-3!">
      <el-select v-model="filters.level" placeholder="全部级别" clearable @change="onFilter" style="width: 160px">
        <el-option label="全部级别" value="" />
        <el-option label="信息" value="info" />
        <el-option label="警告" value="warning" />
        <el-option label="错误" value="error" />
      </el-select>
      <el-select v-model="filters.module" placeholder="全部模块" clearable @change="onFilter" style="width: 160px">
        <el-option label="全部模块" value="" />
        <el-option label="系统" value="system" />
        <el-option label="模拟器" value="simulator" />
        <el-option label="报警" value="alarm" />
        <el-option label="API" value="api" />
        <el-option label="认证" value="auth" />
      </el-select>
    </div>

    <!-- 日志列表 -->
    <div class="bg-white rounded-lg shadow overflow-hidden">
      <el-table :data="logs" v-loading="loading" style="width: 100%" border>
        <el-table-column label="级别" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="levelTagType[row.level as string] || 'info'" size="small">
              {{ row.level }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="module" label="模块" width="120" align="center" />
        <el-table-column prop="message" label="消息" min-width="300" />
        <el-table-column label="时间" width="180" align="center">
          <template #default="{ row }">
            {{ (row.created_at as string)?.substring(0, 19)?.replace('T', ' ') }}
          </template>
        </el-table-column>
        <template #empty>
          <span class="text-gray-400">暂无系统日志</span>
        </template>
      </el-table>

      <!-- 分页 -->
      <div class="flex items-center justify-between px-4 py-3 border-t text-sm text-gray-500">
        <span>共 {{ total }} 条，第 {{ page }} / {{ totalPages }} 页</span>
        <el-pagination background layout="prev, next" :total="total" :page-size="pageSize" :current-page="page"
          @current-change="onPageChange" />
      </div>
    </div>
  </div>
</template>
