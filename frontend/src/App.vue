<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { generate } from './api'
import type { GenerateRequest, GenerateResponse } from './types'

const FREE_LIMIT = 5
const USED_KEY = 'tagcraft_used_free'
const PRO_KEY = 'tagcraft_pro'
const WAITLIST_URL = 'https://docs.google.com/forms/d/e/1FAIpQLSebWLkESa_Hrb789FuukjT5lWik1_q5fKleYWnmf657G7TvnQ/viewform'

const categories = [
  { value: 'jewelry', label: 'Jewelry' },
  { value: 'home', label: 'Home & Living' },
  { value: 'art', label: 'Art & Collectibles' },
  { value: 'clothing', label: 'Clothing' },
  { value: 'craft', label: 'Craft Supplies' },
  { value: 'wedding', label: 'Wedding & Party' },
  { value: 'toys', label: 'Toys & Games' },
  { value: 'other', label: 'Other' },
]

const form = ref<GenerateRequest>({
  product: '',
  keywords: '',
  category: 'jewelry',
})

const result = ref<GenerateResponse | null>(null)
const loading = ref(false)
const errorMsg = ref('')
const copiedKey = ref('')
const upgraded = ref(false)

const usedFree = ref(0)
const isPro = ref(false)

onMounted(() => {
  usedFree.value = Number(localStorage.getItem(USED_KEY) || 0)
  isPro.value = localStorage.getItem(PRO_KEY) === 'true'
  // Stripe 支付成功后跳回 #success，这里解锁
  if (window.location.hash === '#success') {
    localStorage.setItem(PRO_KEY, 'true')
    isPro.value = true
    upgraded.value = true
    window.location.hash = ''
  }
})

const reachedLimit = computed(() => !isPro.value && usedFree.value >= FREE_LIMIT)
const canGenerate = computed(
  () => !loading.value && !reachedLimit.value && form.value.product.trim() !== ''
)
const titleLen = computed(() => result.value?.title.length ?? 0)
const remainingFree = computed(() => Math.max(0, FREE_LIMIT - usedFree.value))

async function handleGenerate() {
  if (!canGenerate.value) return
  errorMsg.value = ''
  result.value = null
  loading.value = true
  try {
    result.value = await generate(form.value)
    if (!isPro.value) {
      usedFree.value += 1
      localStorage.setItem(USED_KEY, String(usedFree.value))
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : 'generation failed'
    if (msg === 'daily free limit reached' && !isPro.value) {
      // 后端 IP 限流已到每日上限，但前端计数可能没同步到 FREE_LIMIT
      // （IP 计数残留 / 同一公网 IP 多人共用 / 后端先于前端拦截）。
      // 强制拉到上限，触发付费墙，避免"后端 429 拦了、前端却不显示入口"的死锁。
      usedFree.value = FREE_LIMIT
      localStorage.setItem(USED_KEY, String(FREE_LIMIT))
    } else {
      // 其他错误（分钟限流 / DeepSeek 失败等）正常展示，不触发付费墙
      errorMsg.value = msg
    }
  } finally {
    loading.value = false
  }
}

// MVP 阶段用 waitlist 收集意向用户邮箱，等切 Paddle 后改回真实 checkout
function handleUpgrade() {
  window.open(WAITLIST_URL, '_blank')
}

async function copy(text: string, key: string) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = key
    setTimeout(() => (copiedKey.value = ''), 1500)
  } catch {
    errorMsg.value = 'copy failed, please copy manually'
  }
}
</script>

<template>
  <div class="page">
    <header class="header">
      <div class="brand">
        <span class="logo">🏷️</span>
        <span class="name">TagCraft</span>
        <span v-if="isPro" class="pro-badge">PRO</span>
      </div>
      <p class="tagline">Etsy SEO titles, tags & descriptions — generated for search.</p>
    </header>

    <main class="container">
      <!-- 升级成功提示 -->
      <div v-if="upgraded" class="banner success">
        🎉 Upgrade successful! You now have unlimited generations.
      </div>

      <!-- 输入区 -->
      <section class="card form-card">
        <h2 class="section-title">Tell us about your product</h2>

        <div class="field">
          <label>Product description</label>
          <textarea
            v-model="form.product"
            placeholder="e.g. Handmade sterling silver ring with a natural turquoise stone, adjustable band, boho style..."
          />
        </div>

        <div class="field">
          <label>Keywords <span class="hint">(comma separated)</span></label>
          <input
            v-model="form.keywords"
            placeholder="e.g. turquoise ring, silver jewelry, boho"
          />
        </div>

        <div class="field">
          <label>Category</label>
          <select v-model="form.category">
            <option v-for="c in categories" :key="c.value" :value="c.value">{{ c.label }}</option>
          </select>
        </div>

        <button class="btn-primary" :disabled="!canGenerate" @click="handleGenerate">
          <span v-if="loading" class="spinner" />
          {{ loading ? 'Generating...' : 'Generate' }}
        </button>

        <p v-if="!isPro" class="counter">
          {{ remainingFree }} of {{ FREE_LIMIT }} free generations left
        </p>
      </section>

      <!-- 错误提示 -->
      <div v-if="errorMsg" class="banner error">{{ errorMsg }}</div>

      <!-- 结果区 -->
      <section v-if="result" class="card result-card">
        <div class="block">
          <div class="block-head">
            <label>Title</label>
            <span class="char-count" :class="{ over: titleLen > 140 }">{{ titleLen }}/140</span>
          </div>
          <p class="block-text">{{ result.title }}</p>
          <button class="btn-copy" @click="copy(result.title, 'title')">
            {{ copiedKey === 'title' ? 'Copied!' : 'Copy' }}
          </button>
        </div>

        <div class="block">
          <div class="block-head">
            <label>Tags</label>
            <span class="char-count">{{ result.tags.length }}/13</span>
          </div>
          <div class="tags">
            <span v-for="tag in result.tags" :key="tag" class="tag">{{ tag }}</span>
          </div>
          <button class="btn-copy" @click="copy(result.tags.join(', '), 'tags')">
            {{ copiedKey === 'tags' ? 'Copied!' : 'Copy all' }}
          </button>
        </div>

        <div class="block">
          <label>Description</label>
          <p class="block-text desc">{{ result.description }}</p>
          <button class="btn-copy" @click="copy(result.description, 'desc')">
            {{ copiedKey === 'desc' ? 'Copied!' : 'Copy' }}
          </button>
        </div>
      </section>

      <!-- 付费墙（MVP 阶段 waitlist，等切 Paddle 后改回真实 checkout） -->
      <section v-if="reachedLimit" class="card paywall">
        <h2>🔒 Free limit reached</h2>
        <p>You've used all {{ FREE_LIMIT }} free generations. TagCraft Pro is coming soon — join the waitlist for early access and a special launch price.</p>
        <button class="btn-upgrade" @click="handleUpgrade">Join Waitlist →</button>
      </section>
    </main>

    <!-- SEO 内容：How it works + Why + FAQ（服务端渲染前靠 JS 渲染，Google 可收录） -->
    <section class="seo">
      <h2>How the Etsy title generator works</h2>
      <ol class="steps">
        <li><strong>Describe your product</strong> — paste what it is, the material, style, and who it's for.</li>
        <li><strong>Add keywords</strong> — the terms buyers actually search for (e.g. "turquoise ring, boho jewelry").</li>
        <li><strong>Generate &amp; copy</strong> — get a title under 140 characters, 13 tags under 20 characters each, and a keyword-rich description to paste into your listing.</li>
      </ol>

      <h2>Why sellers use TagCraft</h2>
      <ul class="points">
        <li><strong>Etsy-compliant, guaranteed</strong> — titles stay under 140 characters and every one of the 13 tags stays under 20 characters, so Etsy never silently rejects them.</li>
        <li><strong>13 long-tail tags</strong> — covering material, style, occasion, and audience, instead of repeating the same keyword.</li>
        <li><strong>Free to start</strong> — 5 free generations per day, no signup.</li>
      </ul>

      <h2>Etsy SEO FAQ</h2>
      <div class="faq">
        <h3>How long can an Etsy title be?</h3>
        <p>Etsy titles are limited to 140 characters. The first 30–40 characters carry the most weight, so TagCraft puts your most important keywords up front.</p>

        <h3>How many tags can I use on Etsy?</h3>
        <p>Etsy gives you 13 tag slots, each up to 20 characters. Tags over 20 characters are rejected, so TagCraft keeps every tag at 20 characters or fewer.</p>

        <h3>What makes good Etsy tags?</h3>
        <p>Long-tail phrases (2–4 words) that match real buyer searches — material, style, use case, occasion, and color — instead of single generic words.</p>

        <h3>Is TagCraft free?</h3>
        <p>Yes — you get 5 free generations per day. A Pro plan is coming soon.</p>
      </div>
    </section>

    <footer class="footer">
      <p>TagCraft — get your Etsy listings found by more buyers.</p>
    </footer>
  </div>
</template>

<style scoped>
.page {
  max-width: 720px;
  margin: 0 auto;
  padding: 32px 20px 60px;
}

.header {
  text-align: center;
  margin-bottom: 32px;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 28px;
  font-weight: 800;
}

.logo {
  font-size: 32px;
}

.name {
  color: var(--accent);
}

.pro-badge {
  background: var(--accent);
  color: #fff;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 20px;
  font-weight: 700;
}

.tagline {
  color: var(--text-muted);
  margin-top: 8px;
  font-size: 15px;
}

.container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  padding: 24px;
}

.section-title {
  font-size: 18px;
  margin-bottom: 18px;
}

.field {
  margin-bottom: 16px;
}

.field label,
.block label {
  display: block;
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 6px;
}

.hint {
  font-weight: 400;
  color: var(--text-muted);
  font-size: 13px;
}

.btn-primary {
  width: 100%;
  background: var(--accent);
  color: #fff;
  padding: 14px;
  font-size: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-hover);
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.4);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.counter {
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
  margin-top: 12px;
}

.banner {
  padding: 12px 16px;
  border-radius: var(--radius);
  font-size: 14px;
  font-weight: 500;
}

.banner.success {
  background: var(--success-soft);
  color: var(--success);
}

.banner.error {
  background: var(--danger-soft);
  color: var(--danger);
}

.result-card {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.block {
  position: relative;
}

.block-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.char-count {
  font-size: 13px;
  color: var(--text-muted);
  font-weight: 500;
}

.char-count.over {
  color: var(--danger);
}

.block-text {
  background: var(--bg);
  border-radius: var(--radius);
  padding: 14px;
  font-size: 15px;
  white-space: pre-wrap;
  word-break: break-word;
}

.block-text.desc {
  min-height: 80px;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  background: var(--bg);
  border-radius: var(--radius);
  padding: 14px;
}

.tag {
  background: var(--accent-soft);
  color: var(--accent-hover);
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
}

.btn-copy {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  padding: 6px 14px;
  font-size: 13px;
  margin-top: 8px;
}

.btn-copy:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.paywall {
  text-align: center;
}

.paywall h2 {
  font-size: 20px;
  margin-bottom: 8px;
}

.paywall p {
  color: var(--text-muted);
  margin-bottom: 18px;
}

.btn-upgrade {
  background: var(--accent);
  color: #fff;
  padding: 14px 32px;
  font-size: 16px;
}

.btn-upgrade:hover {
  background: var(--accent-hover);
}

.footer {
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
  margin-top: 40px;
}

.seo {
  margin-top: 40px;
  padding: 0 4px;
}

.seo h2 {
  font-size: 20px;
  margin: 28px 0 12px;
}

.seo h3 {
  font-size: 15px;
  margin: 16px 0 6px;
}

.seo p,
.seo li {
  color: var(--text-muted);
  line-height: 1.6;
  font-size: 14px;
}

.steps,
.points {
  padding-left: 20px;
  margin: 0 0 8px;
}

.steps li,
.points li {
  margin-bottom: 8px;
}
</style>
