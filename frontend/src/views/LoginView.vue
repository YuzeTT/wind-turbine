<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const form = reactive({
  username: '',
  password: '',
})
const loading = ref(false)
const formRef = ref()

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleLogin() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return
    loading.value = true
    try {
      await auth.login(form.username, form.password)
      const redirect = (route.query.redirect as string) || '/dashboard'
      router.push(redirect)
    } catch (err) {
      ElMessage.error((err as Error).message || '登录失败')
    } finally {
      loading.value = false
    }
  })
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">
    <el-card class="w-96" shadow="always" body-class="p-8">
      <div class="text-center mb-8">
        <el-icon size="48" class="text-blue-500 mb-4"><Cpu /></el-icon>
        <h1 class="text-2xl font-bold text-gray-800 m-0">风电场管理系统</h1>
        <p class="text-sm text-gray-400 mt-2 m-0">东海风电场 · 智能监控平台</p>
      </div>

      <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin" label-position="top">
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            placeholder="用户名"
            size="large"
            :prefix-icon="'User'"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            size="large"
            :prefix-icon="'Lock'"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            class="w-full"
            :loading="loading"
            @click="handleLogin"
          >
            登录
          </el-button>
        </el-form-item>
      </el-form>

      <el-alert type="info" :closable="false" class="mt-4">
        <template #title>
          <div class="text-xs">
            管理员: admin / admin123　操作员: operator / op123456
          </div>
        </template>
      </el-alert>
    </el-card>
  </div>
</template>
