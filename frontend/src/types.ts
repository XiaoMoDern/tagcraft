export interface GenerateRequest {
  product: string
  keywords: string
  category: string
}

export interface GenerateResponse {
  title: string
  tags: string[]
  description: string
}

export interface CheckoutResponse {
  url: string
}

// 历史记录条目：本地 localStorage 存最近 N 次生成，含输入与输出，便于回看
export interface HistoryEntry {
  id: string
  ts: number
  request: GenerateRequest
  response: GenerateResponse
}
