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
