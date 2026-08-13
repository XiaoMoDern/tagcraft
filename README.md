# TagCraft — Etsy SEO Generator

帮 Etsy 手工卖家生成符合平台搜索算法的标题（140 字符）、13 个长尾标签、SEO 产品描述。
壁垒在 Etsy SEO 规则内嵌进 prompt，不在 AI 本身。

> 设计文档：`E:\WorkBuddy\2026-08-13-09-05-19\etsy-seo-mvp-design.md`

## 技术栈

| 层 | 选型 |
|---|---|
| 前端 | Vue3 + TS + Vite（纯 CSS，无 UI 库） |
| 后端 | Go 1.26 + net/http（零第三方依赖，Stripe 走 REST API） |
| AI | DeepSeek API（OpenAI 兼容接口，强制 `response_format: json_object`） |
| 支付 | Stripe Checkout（订阅模式 $19/月） |
| 部署 | 前端 Vercel / 后端 Railway |

## 目录结构

```
tagcraft/
├── backend/
│   ├── main.go        # 入口：路由 + CORS + 启动
│   ├── server.go      # /generate handler + 请求/响应类型
│   ├── deepseek.go    # DeepSeek 调用（含超时、错误处理、env var）
│   ├── prompt.go      # Etsy SEO system prompt（产品核心壁垒）
│   ├── stripe.go      # /create-checkout（直接调 Stripe REST API）
│   └── .env.example
├── frontend/
│   ├── src/
│   │   ├── App.vue    # 表单 + 结果展示 + 付费墙 + Stripe 回跳
│   │   ├── api.ts     # fetch 封装
│   │   ├── types.ts   # 前后端共享类型
│   │   └── style.css  # 全局样式（Etsy 暖色调）
│   ├── vite.config.ts # 开发 proxy → localhost:8787
│   ├── vercel.json    # SPA rewrite
│   └── .env.example   # VITE_API_BASE（生产填 Railway 域名）
└── .gitignore
```

## 本地开发

### 1. 后端

```bash
cd backend
cp .env.example .env
# 填入 DEEPSEEK_API_KEY（必填，https://platform.deepseek.com）
# STRIPE_SECRET_KEY 可先不填，付费墙功能跑不通但不影响生成
go run .
# 服务跑在 http://localhost:8787
```

健康检查：`curl http://localhost:8787/health` → `{"status":"ok"}`

### 2. 前端

```bash
cd frontend
npm install
npm run dev
# 跑在 http://localhost:5173，API 通过 vite proxy 转发到 8787
```

打开 http://localhost:5173 ，填产品描述 + 关键词 + 品类，点 Generate。

### 3. 验证 Stripe（可选）

填好 `STRIPE_SECRET_KEY`（测试 key `sk_test_...`）后，免费次数用完点升级会跳转 Stripe 测试支付页。用测试卡 `4242 4242 4242 4242` 完成支付，跳回 `/#success` 自动解锁。

## 部署

### 后端 → Railway

1. 代码推 GitHub
2. railway.app 新建项目，连仓库（Railway 自动识别根目录 go.mod）
3. 环境变量：
   - `DEEPSEEK_API_KEY` — 必填
   - `STRIPE_SECRET_KEY` — 启用付费必填
   - `STRIPE_SUCCESS_URL` — `https://你的前端域名.vercel.app/#success`
   - `STRIPE_CANCEL_URL` — `https://你的前端域名.vercel.app/`
   - `PORT` — Railway 自动注入，不用填
4. 部署后拿到 `xxx.up.railway.app` 域名

### 前端 → Vercel

1. vercel.com 导入仓库，Root Directory 选 `frontend`
2. 环境变量：`VITE_API_BASE` = `https://xxx.up.railway.app`（Railway 后端域名）
3. 部署后拿到 `xxx.vercel.app` 域名
4. 回 Railway 把 `STRIPE_SUCCESS_URL` / `STRIPE_CANCEL_URL` 改成真实前端域名

> CORS 后端已开 `Access-Control-Allow-Origin: *`，MVP 阶段够用。

## 关键设计决策

1. **后端零第三方依赖**：Stripe 不用 SDK，直接 `net/http` 调 REST API（form-encoded）。设计文档原本用 stripe-go，改成直接调避免依赖、保持"标准库够用"原则。功能等价。
2. **DeepSeek 强制 JSON 输出**：`response_format: {type: "json_object"}`，避免模型输出 markdown 代码块导致解析失败。content 字段需二次 `json.Unmarshal`。
3. **prompt 用英文**：Etsy 是英文市场，输出必须是英文，prompt 也用英文避免模型串语言。
4. **付费墙纯前端**：localStorage 记免费次数（3 次）+ pro 标记。MVP 不做服务端校验，第二版加 webhook。
5. **Stripe 回跳用 hash 路由**：`/#success`，SPA 无需额外路由配置，onMounted 检测 hash 解锁。
