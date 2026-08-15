# TagCraft — Etsy SEO Generator

帮 Etsy 手工卖家生成符合平台搜索算法的标题（140 字符）、13 个长尾标签、SEO 产品描述。
壁垒在 Etsy SEO 规则内嵌进 prompt，不在 AI 本身。

> 设计文档：`E:\WorkBuddy\2026-08-13-09-05-19\etsy-seo-mvp-design.md`

## 技术栈

| 层 | 选型 |
|---|---|
| 前端 | Vue3 + TS + Vite（纯 CSS，无 UI 库） |
| 后端 | Go 1.26 + net/http（零第三方依赖，Stripe 走 REST API） |
| AI | DeepSeek API（默认 `deepseek-v4-flash`，`DEEPSEEK_MODEL` 可覆盖；强制 `response_format: json_object`） |
| 支付 | Stripe Checkout（waitlist 模式，Pro $9/月） |
| 部署 | 前端 Vercel / 后端 Railway |

## 目录结构

```
tagcraft/
├── backend/
│   ├── main.go        # 入口：路由 + CORS + 启动 + loadEnv
│   ├── server.go      # /generate handler + 请求/响应类型 + writeJSON + withCORS
│   ├── deepseek.go    # DeepSeek 调用（默认 v4-flash，DEEPSEEK_MODEL 可覆盖）
│   ├── prompt.go      # Etsy SEO system prompt（7 类禁止声明 + 标签规则）
│   ├── safety.go      # 内容安全：封店级硬词 sanitizeSEO + 声明词 echo 校验 sanitizeClaims
│   ├── safety_test.go # safety.go 单元测试
│   ├── stripe.go      # /create-checkout（直接调 Stripe REST API + tax_code）
│   ├── ratelimit.go   # IP rate limiter（每分钟 5 次 + 每天 5 次免费）
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
2. railway.app 新建项目，连仓库
3. ⚠️ **关键：Settings → Root Directory 填 `backend`**（go.mod 在 `backend/` 子目录下，不填会从仓库根构建，检测到 `frontend/` 跑前端构建报错）
4. 环境变量：
   - `DEEPSEEK_API_KEY` — 必填（https://platform.deepseek.com）
   - `DEEPSEEK_MODEL` — 可选，默认 `deepseek-v4-flash`（旗舰版填 `deepseek-v4-pro`）
   - `STRIPE_SECRET_KEY` — 启用付费必填（Stripe test key `sk_test_...`）
   - `STRIPE_SUCCESS_URL` — `https://你的前端域名.vercel.app/#success`
   - `STRIPE_CANCEL_URL` — `https://你的前端域名.vercel.app/`
   - `PORT` — Railway 自动注入（默认 8080），不用填
5. Deploy → Settings → Networking → **Generate Domain** → 端口填 `8080` → 拿到 `xxx.up.railway.app` 域名
6. 验证：`curl https://xxx.up.railway.app/health` → `{"status":"ok"}`

> **IP rate limit**：`/generate` 有限流，每 IP 每分钟 5 次 + 每天 5 次免费。MVP 阶段无 Stripe webhook，付费用户暂同此限制。重置计数：Railway Redeploy（清内存）。

### 前端 → Vercel

1. vercel.com 导入仓库
2. ⚠️ **Root Directory 选 `frontend`**（不是 `frontend/src`，选 src 会找不到 `package.json` 构建失败）
3. 环境变量：`VITE_API_BASE` = `https://xxx.up.railway.app`（Railway 后端域名，**不带尾斜杠**，带了会变成 `//generate` 导致 404）
   - 新版 Vercel：环境变量在 **Settings → Environments** 子页面（不在 Settings 主菜单），选 Production 环境
4. 部署后拿到 `xxx.vercel.app` 域名
5. 回 Railway 把 `STRIPE_SUCCESS_URL` / `STRIPE_CANCEL_URL` 改成真实前端域名

> CORS 后端已开 `Access-Control-Allow-Origin: *`，MVP 阶段够用。

### Stripe 配置

1. dashboard.stripe.com 注册（可用 Google 登录）
2. 注册时选 **Managed Payments**（推荐，Stripe 帮处理全球税务，每笔 +3.5% 费用）
3. Developers → API Keys → 拿 `sk_test_...`（test mode，不涉及真钱）
4. 填到 Railway Variables 的 `STRIPE_SECRET_KEY`
5. 测试卡：`4242 4242 4242 4242` / 任意未来日期 / 任意 CVC / 任意美国地址

> ⚠️ **中国个人开发者注意**：Stripe live mode（真收钱）需要海外实体。MVP 阶段用 test mode 验证流程，真收钱时换 **Paddle / Lemon Squeezy**（merchant of record，支持中国个人）或注册海外公司。
>
> ⚠️ **Managed Payments 要求 tax_code**：`stripe.go` 已加 `tax_code = txcd_10103001`（SaaS）。如果换其他产品类型，去 https://stripe.com/docs/tax/tax-codes 找对应 code。

## 关键设计决策

1. **后端零第三方依赖**：Stripe 不用 SDK，直接 `net/http` 调 REST API（form-encoded）。设计文档原本用 stripe-go，改成直接调避免依赖、保持"标准库够用"原则。功能等价。
2. **DeepSeek 强制 JSON 输出**：`response_format: {type: "json_object"}`，避免模型输出 markdown 代码块导致解析失败。content 字段需二次 `json.Unmarshal`。
3. **prompt 用英文**：Etsy 是英文市场，输出必须是英文，prompt 也用英文避免模型串语言。
4. **付费墙纯前端**：localStorage 记免费次数（5 次）+ pro 标记。MVP 不做服务端校验，第二版加 webhook。
5. **Stripe 回跳用 hash 路由**：`/#success`，SPA 无需额外路由配置，onMounted 检测 hash 解锁。
6. **IP rate limit 防刷**：`/generate` 加 middleware，每 IP 每分钟 5 次（防 burst）+ 每天 5 次免费（防白嫖兜底）。滑动窗口 + 滚动 24h + goroutine 定时清理过期 IP。MVP 阶段无 webhook，付费用户暂同此限制。
7. **内容安全三层校验**：① prompt 内嵌 7 类禁止声明（预防）② `safety.go` 的 `sanitizeSEO` 硬词表剔除封店级词（部落/品牌/医疗，无条件）③ `sanitizeClaims` 声明词 echo 校验（绿色/安全/材质声明只有卖家逐字提过才保留）。三层各自独立，封店级词保证不落地。

## 部署踩坑记录（2026-08-13 首次部署）

| 坑 | 现象 | 修复 |
|---|---|---|
| Railway Root Directory 没填 | 从仓库根构建，检测到 frontend 跑 `npm run build` 报 TS 错误 | Settings → Root Directory 填 `backend` |
| Vercel Root Directory 选错 | 选了 `frontend/src`，找不到 `package.json` 构建失败 | 选 `frontend`（不是 src 子目录） |
| TS6310 构建错误 | `tsconfig.node.json` 同时设 `composite: true` + `noEmit: true`，vue-tsc -b 报错 | `noEmit: true` 改 `false`（commit `04cc100`） |
| VITE_API_BASE 找不到入口 | 新版 Vercel 把环境变量移到 Environments 子页面 | Settings → Environments → Production |
| Vercel 前端 405 | VITE_API_BASE 留空，POST /generate 被 vercel.json rewrite 到 index.html，静态文件只支持 GET | 填 Railway 后端域名 |
| Railway Generate Domain 端口 | 不知道填什么端口 | 8080（Railway 自动注入 PORT=8080） |
| Stripe tax_code missing | Managed Payments 要求 line_items 必须有 tax_code | `stripe.go` 加 `tax_code = txcd_10103001`（commit `fface21`） |
| 国内 GitHub push 超时 | HTTPS 443 直连超时，配代理 7890 Clash 没开 | SSH over 443 + Host 别名 + 专用 key（见下） |
| SSH config 权限错误 | `Bad owner or permissions on config` | `icacls /inheritance:r /grant:r "WSZ:F" /grant:r "SYSTEM:F"` |
| SSH key 加到 Deploy keys | `marked as read only`，push 失败 | 删 deploy key，加到账号级 SSH keys |

### 国内 GitHub push 解决方案（SSH over 443 + Host 别名）

国内访问 GitHub HTTPS 443 经常超时，SSH 22 也常被墙。最稳方案：

```bash
# 1. 生成专用 key（不影响其他项目）
ssh-keygen -t ed25519 -C "tagcraft" -f ~/.ssh/tagcraft_ed25519 -N ""

# 2. 配 ~/.ssh/config（Host 别名隔离，不影响 github.com 其他仓库）
cat > ~/.ssh/config << 'EOF'
Host github-tagcraft
  HostName ssh.github.com
  Port 443
  User git
  IdentityFile ~/.ssh/tagcraft_ed25519
  IdentitiesOnly yes
EOF
chmod 600 ~/.ssh/config  # Windows: icacls "C:\Users\WSZ\.ssh\config" /inheritance:r /grant:r "WSZ:F" /grant:r "SYSTEM:F"

# 3. 复制公钥到 GitHub 账号级 SSH keys（不是仓库 Deploy keys）
cat ~/.ssh/tagcraft_ed25519.pub
# → 去 https://github.com/settings/ssh/new 粘贴，Key type 选 Authentication Key

# 4. 改 remote 用 Host 别名
git remote set-url origin git@github-tagcraft:XiaoMoDern/tagcraft.git

# 5. 验证
ssh -T git@github-tagcraft
# 期望: Hi XiaoMoDern! You've successfully authenticated...（账号名，不是仓库名）
```
