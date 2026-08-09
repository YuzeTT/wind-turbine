<script setup lang="ts">
import { ref, computed } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const isCollapse = ref(false)

const menuItems = computed(() => {
  const items = [
    { name: 'dashboard', label: '总览看板', icon: 'DataLine' },
    { name: 'turbines', label: '风机列表', icon: 'Operation' },
    { name: 'alarms', label: '报警管理', icon: 'Warning' },
    { name: 'operation-logs', label: '操作日志', icon: 'Document' },
    { name: 'system-logs', label: '系统日志', icon: 'Tickets' },
  ]
  if (auth.isAdmin) {
    items.push({ name: 'users', label: '用户管理', icon: 'User' })
  }
  return items
})

const activeMenu = computed(() => router.currentRoute.value.name as string)

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <el-container class="h-screen">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '210px'" class="transition-all duration-300">
      <div class="h-full flex flex-col" style="background-color: #001529;">
        <!-- Logo -->
        <div class="h-14 flex items-center justify-center text-white border-b border-white/10">
          <el-icon size="24">
            <Cpu />
          </el-icon>
          <span v-if="!isCollapse" class="text-base font-bold whitespace-nowrap ml-2!">风电监控系统</span>
        </div>

        <!-- 菜单 -->
        <el-menu :default-active="activeMenu" :collapse="isCollapse" background-color="#001529" text-color="#ffffffa0"
          active-text-color="#409eff" class="flex-1 border-r-0!" @select="(name: string) => router.push({ name })">
          <el-menu-item v-for="item in menuItems" :key="item.name" :index="item.name">
            <el-icon>
              <component :is="item.icon" />
            </el-icon>
            <template #title>{{ item.label }}</template>
          </el-menu-item>
        </el-menu>

        <!-- 折叠按钮 -->
        <div class="p-2 border-t border-white/10">
          <el-button text class="w-full text-white/70" @click="isCollapse = !isCollapse">
            <el-icon>
              <Fold v-if="!isCollapse" />
              <Expand v-else />
            </el-icon>
          </el-button>
        </div>
      </div>
    </el-aside>

    <!-- 主区域 -->
    <el-container>
      <el-header class="flex items-center justify-between border-b border-gray-200 bg-white px-6">
        <h2 class="text-lg font-semibold text-gray-800 m-0">
          {{ router.currentRoute.value.meta.title || '首页' }}
        </h2>
        <div class="flex items-center gap-4">
          <div text-sm>{{ auth.user?.nickname }}</div>
          <el-tag :type="auth.isAdmin ? 'danger' : 'success'" effect="plain" size="small">
            {{ auth.isAdmin ? '管理员' : '操作员' }}
          </el-tag>
          <el-dropdown>
            <el-button text>
              <el-icon>
                <Setting />
              </el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleLogout">
                  <el-icon>
                    <SwitchButton />
                  </el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="bg-gray-50 p-6 overflow-auto">
        <RouterView />
      </el-main>
    </el-container>
  </el-container>
</template>
