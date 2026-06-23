# Store Mind 顾客端微信小程序

> 小王 · 数字店员 — 无人超市顾客问答体验

## 技术栈

- **框架**: Taro 4.x + Vue 3 + TypeScript
- **状态管理**: Pinia
- **构建目标**: 微信小程序 (WeChat Mini Program)
- **测试**: Vitest + @vue/test-utils
- **样式**: SCSS

## 本地开发

### 环境准备

- Node.js 18–20（Taro 4.x 当前不支持 Node 22+）
- 推荐使用 nvm 或 fnm 管理 Node 版本
- ~~微信开发者工具~~（用于预览和调试）

### 第一步：安装依赖

```bash
cd frontend
npm install --legacy-peer-deps
```

### 第二步：启动开发模式

```bash
npm run dev:weapp
```

编译成功后会生成 `frontend/dist/` 目录。开发模式下会自动监听文件变更并增量编译。

### 第三步：导入微信开发者工具

1. 打开**微信开发者工具**
2. 点击左侧 **「+」** 或右上角 **「导入项目」**
3. 填写项目信息：

| 字段 | 内容 |
|---|---|
| 项目名称 | 任意命名，如 `StoreMind 顾客端` |
| 目录 | 选择 `frontend/dist` |
| AppID | 选择 **「测试号」** 或填入自有 AppID |
| 开发模式 | **小程序** |
| 后端服务 | **不使用云服务** |

4. 点击 **「确定」** 进入开发者界面

> 💡 如果是第一次使用，需要先注册微信公众平台账号并完成小程序开发者认证：
> 访问 [https://mp.weixin.qq.com](https://mp.weixin.qq.com) 注册 → 选择「小程序」类型 → 获取 AppID。

### 第四步：配置网络与代理

小程序默认要求所有 API 请求走 HTTPS 域名。本地开发时需要：

1. 在微信开发者工具顶部菜单 → **「详情」** → **「本地设置」**
2. 勾选 **「不校验合法域名、web-view（业务域名）、TLS 版本以及 HTTPS 证书」**
3. 如果后端在本地运行，还需要配置代理（可选）：
   - 用内网穿透工具（如 ngrok、frp）将 `localhost:8080` 暴露为 HTTPS 公网地址
   - 或者在代码 `src/utils/env.ts` 中直接使用 `http://localhost:8080`（工具中已勾选不校验域名时可用 `http://` 地址）

### 第五步：在模拟器中验证

成功导入后，开发者工具会自动加载门店首页（`pages/store/index`）：

1. 确认能看到「阳光便利店 No.3」门店卡片
2. 点击「问小王」跳转到聊天页面
3. 尝试输入文字发送消息

---

### 生产构建

```bash
npm run build:weapp
```

### 类型检查

```bash
npm run typecheck
```

### 运行测试

```bash
npm test          # 运行所有测试（当前 30 个测试全部通过）
npm run test:watch  # 监听模式
```

## 项目结构

```
frontend/
├── config/              # Taro 构建配置
│   ├── index.ts         # 主配置
│   ├── dev.ts           # 开发环境
│   └── prod.ts          # 生产环境
├── src/
│   ├── app.ts           # 应用入口 (注册 Pinia)
│   ├── app.config.ts    # 页面路由配置
│   ├── app.scss         # 全局样式 & CSS 变量
│   ├── pages/
│   │   ├── store/       # 门店首页
│   │   │   ├── index.vue
│   │   │   ├── index.config.ts
│   │   │   └── index.scss
│   │   └── chat/        # 聊天页面
│   │       ├── index.vue
│   │       ├── index.config.ts
│   │       └── index.scss
│   ├── components/
│   │   ├── store/       # 门店组件
│   │   │   ├── StoreGreeting.vue
│   │   │   └── StoreRecommendationList.vue
│   │   ├── chat/        # 聊天组件
│   │   │   ├── ChatHeader.vue
│   │   │   ├── ZoneBanner.vue
│   │   │   ├── MessageList.vue
│   │   │   ├── MessageBubble.vue
│   │   │   ├── AnswerCard.vue
│   │   │   ├── GuidanceChips.vue
│   │   │   ├── FeedbackBar.vue
│   │   │   └── ChatInput.vue
│   │   └── common/      # 通用组件
│   │       ├── StatusPill.vue
│   │       ├── InlineNotice.vue
│   │       └── EmptyState.vue
│   ├── services/
│   │   ├── api.ts       # HTTP 客户端 + 错误规范化
│   │   └── customerQa.ts # 顾客问答 API 封装
│   ├── stores/
│   │   ├── session.ts   # 会话状态管理
│   │   └── chat.ts      # 聊天状态管理
│   ├── types/
│   │   └── customerQa.ts # 共享类型定义
│   └── utils/
│       ├── env.ts       # 环境配置
│       ├── qr.ts        # QR 码解析
│       └── storage.ts   # 本地存储封装
├── package.json
├── tsconfig.json
└── vitest.config.ts
```

## 页面说明

### 门店首页 (`pages/store/index`)

顾客进入门店后的入口页面，展示：

- 门店名称和地址
- 小王问候卡片（在线状态 + CTA）
- 今日推荐列表（促销活动、新品、热销）
- 扫码购物功能

### 聊天页 (`pages/chat/index`)

唯一的对话页面，通过 query 参数区分入口模式：

| entry 值 | 说明 |
|---|---|
| `first_open` | 首次打开，展示破冰问候 + 引导问题 |
| `zone_scan` | 扫码货架，展示区域 banner + 商品列表 |
| `promo` | 活动入口，展开活动详情 |
| `resume` | 恢复历史会话，显示上下文桥接 |

## API 配置

默认 API 地址为 `http://localhost:8080`，可在 `src/utils/env.ts` 中修改。

开发时若需连接远程后端，修改 `API_BASE_URL` 为内网穿透地址即可。

## 设计规范

- **暗色主基调**：`#1c1b18`，模拟"耳边轻语"的私密服务感
- **品牌暖色**：`#c9863a`，作为交互强调色
- **中文字体**：系统原生 PingFang SC / Microsoft YaHei
- **设计原型参考**：`docs/design/prototypes/v3.html`
