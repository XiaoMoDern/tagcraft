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
  const res = await fetch(`${BASE}/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function createCheckout(): Promise<CheckoutResponse> {
  const res = await fetch(`${BASE}/create-checkout`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}
