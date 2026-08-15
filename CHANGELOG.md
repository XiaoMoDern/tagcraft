# Changelog

TagCraft 的重要变更记录，按时间倒序。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

> 每个条目末尾的短 hash 是对应 commit，可用 `git show <hash>` 查看完整 diff。

## 2026-08-15

### Added
- **内容安全三层校验**（`backend/safety.go`）：
  - `sanitizeSEO`：封店级硬词表（原住民/部落声明、大牌 IP、医疗疗效），命中即从输出剔除
  - `sanitizeClaims`：声明词 echo 校验——`vegan` / `sterling silver` / `solid gold` 等声明词，只有**逐字出现在卖家输入里**才保留，否则剔除
- **单元测试**（`backend/safety_test.go`）：5 个用例覆盖硬词剔除、颜色词不误杀、词边界、声明词剔除/保留

### Changed
- `prompt.go`：`[Safety]` 从 4 条扩成 **7 类禁止声明**（医疗疗效 / 产地部落 / 材质虚标 / 安全认证 / 品牌 IP / 濒危材料 / 绿色宣称）
- `deepseek.go`：默认模型 `deepseek-chat` → **`deepseek-v4-flash`**（更便宜），可用环境变量 `DEEPSEEK_MODEL` 覆盖（如 `deepseek-v4-pro`）
- `server.go`：标签截断改用 `utf8.RuneCountInString` + 去尾部停用词，并接线 `sanitizeSEO` + `sanitizeClaims` 两层校验

### Fixed
- 标签超长被硬切成碎片（如 `guest book for`、`personalized guest`）——改为「词边界 + 去停用词」截断
- 珠宝类输出 `native american` 标签的封店级法律风险——安全护栏拦截
- 免费次数不按天重置——前端 localStorage 永久计数 + 后端滚动 24h，均改为「自然日重置」（前端按本地日、后端按 UTC 日）

## 2026-08-14

### Fixed
- 后端每日免费额度与前端对齐（3 → 5）`decaef1`
- 后端返回「每日免费额度已用完」时，前端正确显示付费墙 `8dea6ff`

## 2026-08-13

### Added
- **TagCraft MVP**：Etsy SEO 生成器（≤140 字符标题 + 13 长尾标签 + SEO 描述）`69541cd`
- `/generate` IP 限流（每分钟 5 次 + 每日免费额度）`d4ab509`

### Changed
- 付费墙改为 waitlist 模式，定价 $19 → **$9**，免费层 5 次/天 `c72a45c`

### Fixed
- `tsconfig.node.json` noEmit 与 composite 冲突（TS6310）`04cc100`
- Stripe Managed Payments 缺 `tax_code` `fface21`

### Docs / Chore
- 补 README 部署踩坑记录 + 国内 SSH over 443 push 方案 `5a8a5ec`
- 清理追踪的构建产物 `7f8ca60`、忽略 `.workbuddy/` `fb38dfa`
