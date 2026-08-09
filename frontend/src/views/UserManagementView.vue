<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listUsers, updateUserStatus, register } from '@/api/auth'

const users = ref<Record<string, unknown>[]>([])
const loading = ref(true)

// 注册弹窗
const showRegisterModal = ref(false)
const registerForm = ref({
  username: '',
  password: '',
  nickname: '',
  role: 'operator',
})

async function fetchUsers() {
  loading.value = true
  try {
    const res = await listUsers()
    users.value = res.data
  } finally {
    loading.value = false
  }
}

async function toggleStatus(user: Record<string, unknown>) {
  const newStatus = user.status === 'active' ? 'disabled' : 'active'
  try {
    await updateUserStatus(user.id as number, newStatus as string)
    await fetchUsers()
  } catch (err) {
    ElMessage.error((err as Error).message)
  }
}

function openRegisterModal() {
  registerForm.value = { username: '', password: '', nickname: '', role: 'operator' }
  showRegisterModal.value = true
}

async function submitRegister() {
  try {
    await register(registerForm.value)
    showRegisterModal.value = false
    await fetchUsers()
  } catch (err) {
    ElMessage.error((err as Error).message)
  }
}

onMounted(fetchUsers)
</script>

<template>
  <div class="space-y-4">
    <div class="flex justify-between items-center">
      <h2 class="text-lg font-semibold text-gray-700">用户列表</h2>
      <el-button type="primary" @click="openRegisterModal">+ 注册新用户</el-button>
    </div>

    <div class="bg-white rounded-lg shadow overflow-hidden">
      <el-table :data="users" v-loading="loading" style="width: 100%" border>
        <el-table-column prop="id" label="ID" width="80" align="left" />
        <el-table-column prop="username" label="用户名" width="150" align="left" />
        <el-table-column prop="nickname" label="昵称" width="150" align="left" />
        <el-table-column label="角色" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'" size="small">
              {{ row.role === 'admin' ? '管理员' : '操作员' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '正常' : '已禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="注册时间" width="150" align="left">
          <template #default="{ row }">
            {{ (row.created_at as string)?.substring(0, 10) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-button
              :type="row.status === 'active' ? 'danger' : 'success'"
              link
              size="small"
              @click="toggleStatus(row)"
            >
              {{ row.status === 'active' ? '禁用' : '启用' }}
            </el-button>
          </template>
        </el-table-column>
        <template #empty>
          <span class="text-gray-400">暂无用户数据</span>
        </template>
      </el-table>
    </div>

    <!-- 注册弹窗 -->
    <el-dialog v-model="showRegisterModal" title="注册新用户" width="420px">
      <el-form :model="registerForm" label-width="80px">
        <el-form-item label="用户名">
          <el-input v-model="registerForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="registerForm.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="registerForm.nickname" placeholder="请输入昵称" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="registerForm.role" style="width: 100%">
            <el-option label="操作员" value="operator" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="showRegisterModal = false">取消</el-button>
          <el-button type="primary" @click="submitRegister">注册</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>
