# 🌬️ 风电场智能监控系统

> 30 台风力发电机的实时监控、故障管理、大屏看板系统

## 项目概览

本项目是一个风电场管理系统的完整演示，包含后端 API 服务、实时数据模拟器、WebSocket 推送和前端大屏看板。系统模拟了 30 台风力发电机在不同风速条件下的运行状态，支持远程启停控制、故障自动上报与自动排障、实时数据可视化等功能。

### 核心特性

- **实时模拟**：30 台风机数据每 3 秒更新，风速随机游走驱动功率曲线计算
- **WebSocket 逐台分发**：避免大批量瞬时传输，每台风机独立推送（80ms 错开）
- **功率缓起缓停**：启停时功率以 3% 步长平滑变化（后端计算，前端刷新不丢数据）
- **故障自动排障**：故障 30-120 秒后自动恢复，维护状态无活跃报警后自动转待机
- **JWT 鉴权**：管理员/操作员双角色，写操作需管理员权限
- **断连检测**：5 秒心跳 + 15 秒超时 + 全屏断连遮罩
- **SVG 风机动画**：叶片旋转随转速变化，颜色随状态变化
- **竖向发电进度条**：实时显示功率百分比，绿满红低

---

## 技术架构

```
┌─────────────────────────────────────────────────────┐
│                    前端 (Vue 3)                      │
│  Element Plus + Tailwind CSS 4 + Pinia + Vue Router  │
│  axios 封装 → REST API 调用                           │
│  WebSocket → 实时增量更新 + 心跳检测                   │
├─────────────────────────────────────────────────────┤
│                  通信层 (HTTP + WS)                   │
├─────────────────────────────────────────────────────┤
│                   后端 (Go + Gin)                     │
│  JWT 鉴权 → 路由中间件                                │
│  GORM → SQLite (pure Go, 无 CGO)                     │
│  模拟器 goroutine → 风速游走/功率曲线/故障/排障        │
│  WebSocket Hub → 逐台分发 + 心跳                      │
└─────────────────────────────────────────────────────┘
```

### 后端技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.26+ | 运行时 |
| Gin | 1.12 | HTTP 框架 |
| GORM | 1.31 | ORM |
| glebarez/sqlite | 1.11 | 纯 Go SQLite 驱动（无 CGO） |
| gorilla/websocket | 1.5 | WebSocket |
| golang-jwt/jwt/v5 | 5.3 | JWT 认证 |
| bcrypt | - | 密码哈希 |

### 前端技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Vue | 3.5 | 框架 |
| Vite | 8.1 | 构建 |
| TypeScript | 6.0 | 类型安全 |
| Element Plus | 2.14 | UI 组件（按需导入） |
| Tailwind CSS | 4.3 | 原子化 CSS |
| Pinia | 4.0 | 状态管理 |
| Vue Router | 5.2 | 路由 + 守卫 |
| axios | 1.19 | HTTP 请求 |

---

## 实现方案

### 模拟器核心逻辑

模拟器是后端的核心，以 3 秒为周期运行以下流程：

1. **风速随机游走**：每台风机风速在当前值上 ±1.0 m/s 波动，范围 0-30 m/s
2. **功率曲线计算**：切入 3 m/s、额定 12 m/s、切出 25 m/s，立方关系近似
3. **功率缓起缓停**：`displayPower` map 跨 tick 持久跟踪，3% 步长追赶目标值，到达后 ±2% 微浮动
4. **状态自动流转**：
   - `running` + 风速 >25 → `weather_stop`（自动停机）
   - `weather_stop` + 风速 <23 → `standby`（恢复待机）
   - `standby` + 风速 3-25 → `running`（自动启动）
   - `running` 0.5%/tick 概率 → 触发随机故障
   - `fault` 30-120 秒后 → 自动排障 → `standby`（等待手动启动）
   - `maintenance` + 无活跃报警 → 10-20 秒后 → `standby`
5. **WebSocket 分发**：30 台风机在 goroutine 中逐台推送，80ms 间隔错开
6. **心跳推送**：每 5 秒广播 heartbeat 消息

### 功率曲线公式

```
v < 3 m/s        → P = 0（切入前）
3 ≤ v < 12 m/s   → P = P_rated × ((v-3)/(12-3))³（立方曲线）
12 ≤ v ≤ 25 m/s  → P = P_rated（额定功率）
v > 25 m/s       → P = 0（切出保护）
```

### 鉴权方案

- JWT Bearer Token，24 小时有效期
- `AuthRequired` 中间件：所有 API 需登录
- `AdminRequired` 中间件：写操作（启停、报警上报、用户管理）需管理员
- WebSocket 通过 URL 参数 `?token=xxx` 鉴权
- 前端 axios 拦截器自动带 token，401 自动跳登录

### 前端实时数据流

```
页面加载 → REST 拉取全量风机列表（初始数据）
         → WebSocket 连接（带 token）
         → 逐台接收 turbine_update 消息 → 增量更新 displayTurbines[idx]
         → 每 5 秒接收 heartbeat → 更新 lastHeartbeat
         → 15 秒无心跳 → isConnected = false → 断连遮罩
```

---

## 项目结构

```
wind_turbine/
├── backend/                    # Go 后端
│   ├── main.go                 # 入口
│   ├── api/                    # HTTP 处理器
│   │   ├── auth.go             # 登录/注册/用户管理
│   │   ├── turbine.go          # 风机 CRUD
│   │   ├── alarm.go            # 报警 CRUD
│   │   ├── oplog.go            # 操作日志
│   │   ├── syslog.go           # 系统日志
│   │   ├── dashboard.go        # 看板数据
│   │   ├── ws.go               # WebSocket Handler
│   │   └── response.go         # 统一响应格式
│   ├── model/model.go          # GORM 模型 + 常量
│   ├── db/db.go                # 数据库初始化 + 种子数据
│   ├── sim/sim.go              # 数据模拟器
│   ├── ws/hub.go               # WebSocket 连接管理 + 心跳
│   ├── middleware/auth.go      # JWT 中间件
│   └── router/router.go        # 路由注册
│
├── frontend/                   # Vue3 前端
│   ├── src/
│   │   ├── api/                # axios 封装 + API 模块
│   │   │   ├── request.ts      # axios 实例 + 拦截器
│   │   │   ├── auth.ts         # 认证 API
│   │   │   ├── turbine.ts      # 风机 API
│   │   │   ├── alarm.ts        # 报警 API
│   │   │   ├── oplog.ts        # 操作日志 API
│   │   │   ├── syslog.ts       # 系统日志 API
│   │   │   ├── dashboard.ts    # 看板 API
│   │   │   └── websocket.ts    # WebSocket 封装
│   │   ├── stores/auth.ts      # Pinia 认证 store
│   │   ├── router/index.ts     # 路由 + 守卫
│   │   ├── layouts/MainLayout.vue  # 侧边栏布局
│   │   ├── components/
│   │   │   └── WindTurbineIcon.vue  # SVG 风机动画
│   │   ├── views/
│   │   │   ├── LoginView.vue
│   │   │   ├── DashboardView.vue       # 大屏看板
│   │   │   ├── TurbineListView.vue     # 风机列表
│   │   │   ├── TurbineDetailView.vue   # 单机详情
│   │   │   ├── AlarmView.vue           # 报警管理
│   │   │   ├── OperationLogView.vue    # 操作日志
│   │   │   ├── SystemLogView.vue       # 系统日志
│   │   │   └── UserManagementView.vue  # 用户管理
│   │   ├── utils/format.ts     # 时间格式化
│   │   └── assets/             # 样式
│   └── vite.config.ts          # Vite + Element Plus 按需导入
│
└── README.md
```

---

## 运行指南

### 环境要求

- Go 1.21+
- Node.js 22+
- Git

### 1. 克隆仓库

```bash
git clone git@github.com:YuzeTT/wind-turbine.git
cd wind-turbine
```

### 2. 启动后端

```bash
cd backend

# 下载依赖（国内需设置代理）
go env -w GOPROXY=https://goproxy.cn,direct
go mod tidy

# 编译
go build -o wind_turbine_backend.exe .

# 运行
./wind_turbine_backend.exe
```

后端启动后：
- HTTP 服务：`http://localhost:8080`
- WebSocket：`ws://localhost:8080/ws`
- SQLite 数据库：`wind_turbine.db`（自动创建，含种子数据）

### 3. 启动前端

```bash
cd frontend

# 安装依赖
npm install

# 开发模式
npm run dev
```

浏览器访问 `http://localhost:5173`

### 4. 生产构建

```bash
cd frontend
npm run build
# 产物在 dist/ 目录
```

### 默认账户

| 用户名 | 密码 | 角色 | 权限 |
|--------|------|------|------|
| admin | admin123 | 管理员 | 全部操作 |
| operator | op123456 | 操作员 | 只读 |

> 删除 `wind_turbine.db` 文件重启后端可重置为默认密码。

---

## API 概览

### 风机管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/v1/turbines` | 风机列表 | 登录 |
| GET | `/api/v1/turbines/:id` | 风机详情 | 登录 |
| PUT | `/api/v1/turbines/:id/status` | 切换状态 | 管理员 |

### 报警管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/v1/alarms` | 报警列表 | 登录 |
| POST | `/api/v1/alarms` | 上报故障 | 管理员 |
| PUT | `/api/v1/alarms/:id/resolve` | 处理报警 | 管理员 |
| GET | `/api/v1/alarms/stats` | 报警统计 | 登录 |

### 看板数据

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/dashboard/overview` | 总览数据 |
| GET | `/api/v1/dashboard/status-distribution` | 状态分布饼图 |
| GET | `/api/v1/dashboard/power-trend` | 24h 功率趋势 |
| GET | `/api/v1/dashboard/availability` | 可用性图表 |
| GET | `/api/v1/dashboard/wind-rose` | 风玫瑰图 |
| GET | `/api/v1/dashboard/daily-energy` | 7 日发电量 |
| GET | `/api/v1/dashboard/map` | 风机地图 |

### WebSocket

```
ws://localhost:8080/ws?token=<JWT_TOKEN>
```

消息类型：

| type | 说明 | 频率 |
|------|------|------|
| `welcome` | 连接成功 | 连接时一次 |
| `turbine_update` | 单台风机数据 | 逐台 80ms 错开 |
| `heartbeat` | 心跳 | 每 5 秒 |

---

## 页面功能

| 页面 | 功能 |
|------|------|
| 登录 | 用户名/密码登录，JWT 认证 |
| 总览看板 | 6 指标卡片 + 状态分布 + 功率趋势 + 日发电量 + 实时风机状态网格 |
| 风机列表 | 全量表格，支持状态筛选，WS 实时更新 |
| 风机详情 | 单机参数 + 近期报警 + 7 天统计 + 状态切换 |
| 报警管理 | 报警列表 + 统计 + 手动上报 + 处理弹窗 |
| 操作日志 | 操作记录列表 + 手动新增操作 |
| 系统日志 | 系统运行日志，支持级别/模块筛选 |
| 用户管理 | 用户列表 + 启用/禁用 + 注册新用户（管理员） |

---

## License

MIT
