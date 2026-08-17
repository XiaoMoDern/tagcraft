import type { GenerateRequest, GenerateResponse, CheckoutResponse } from './types'

// 开发时走 vite proxy（同源），生产时通过 VITE_API_BASE 指向后端域名
const BASE = import.meta.env.VITE_API_BASE || ''

async function parseError(res: Response): Promise<string> {
  try {
    const data = await res.json()
    return data.error || `HTTP ${res.status}`
  } catch {
    return `HTTP ${res.status}`
  }
}

export async function generate(req: GenerateRequest): Promise<GenerateResponse> {
  // 后端偶发冷启动（Railway 睡眠唤醒）+ DeepSeek 推理，最长可能 30-60s。
  // 加 60s 超时：避免之前"无响应"时浏览器无限等待，超时给明确提示而非挂死。
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 60_000)
  try {
    const res = await fetch(`${BASE}/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
      signal: controller.signal,
    })
    if (!res.ok) throw new Error(await parseError(res))
    return res.json()
  } catch (e) {
    if (e instanceof DOMException && e.name === 'AbortError') {
      throw new Error('timed out after 60s — the server may be waking from sleep, try again in a few seconds')
    }
    throw e
  } finally {
    clearTimeout(timer)
  }
}

export async function createCheckout(): Promise<CheckoutResponse> {
  const res = await fetch(`${BASE}/create-checkout`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}
