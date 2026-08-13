package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const stripeCheckoutURL = "https://api.stripe.com/v1/checkout/sessions"

type stripeSessionResponse struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// handleCreateCheckout 创建一个 Stripe Checkout Session，返回跳转 URL。
// MVP 阶段直接调 Stripe REST API，不引 stripe-go SDK，保持零依赖。
// 订阅模式：$19/月。成功后跳回前端 #success，前端 localStorage 标记解锁。
func handleCreateCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	if apiKey == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "stripe not configured"})
		return
	}

	successURL := os.Getenv("STRIPE_SUCCESS_URL")
	if successURL == "" {
		successURL = "http://localhost:5173/#success"
	}
	cancelURL := os.Getenv("STRIPE_CANCEL_URL")
	if cancelURL == "" {
		cancelURL = "http://localhost:5173/"
	}

	// Stripe API 接收 form-encoded body（不是 JSON）
	// $19/月 订阅，用 price_data 内联定价，省去预建 Product/Price 的步骤
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", "usd")
	form.Set("line_items[0][price_data][unit_amount]", "900") // 900 美分 = $9.00
	form.Set("line_items[0][price_data][recurring][interval]", "month")
	form.Set("line_items[0][price_data][product_data][name]", "TagCraft Pro")
	// Managed Payments 要求 product tax_code（SaaS 用 txcd_10103001）
	// https://stripe.com/docs/tax/tax-codes
	form.Set("line_items[0][price_data][product_data][tax_code]", "txcd_10103001")

	req, err := http.NewRequest(http.MethodPost, stripeCheckoutURL, strings.NewReader(form.Encode()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create checkout request"})
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "failed to contact stripe"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp stripeSessionResponse
		_ = json.Unmarshal(body, &errResp)
		msg := "stripe error"
		if errResp.Error != nil {
			msg = errResp.Error.Message
		}
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: msg})
		return
	}

	var session stripeSessionResponse
	if err := json.Unmarshal(body, &session); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to parse stripe response"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"url": session.URL,
	})
}
