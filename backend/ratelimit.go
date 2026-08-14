package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// IP rate limiter —— 防止 DeepSeek 额度被刷。
// 两层限制：
//   1. 每分钟 5 次：防 burst（脚本一秒打 100 次）
//   2. 每天 5 次：防白嫖（普通人无限免费调用）
//
// MVP 阶段没有 Stripe webhook，后端不知道谁是付费用户，
// 所以这个限制对所有人都一样。等 v2 加了 webhook 再按 Pro 状态放宽。
//
// 注意：IP 限流的固有缺陷 —— 同一公网 IP 下多个用户（公司/学校/NAT）
// 会被一起限 5 次。MVP 阶段接受这个取舍。

const (
	rateLimitPerMinute = 5 // 每 IP 每分钟最多 5 次
	freeLimitPerDay    = 5 // 每 IP 每天最多 5 次免费
)

type ipStats struct {
	minuteWindow []time.Time // 最近一分钟的请求时间戳（滑动窗口）
	dayCount     int         // 今天已用的次数
	dayResetAt   time.Time   // 今天计数重置时间（滚动 24 小时）
	lastSeen     time.Time   // 最后一次请求时间（用于清理过期 IP）
}

type rateLimiter struct {
	mu    sync.Mutex
	stats map[string]*ipStats
}

func newRateLimiter() *rateLimiter {
	rl := &rateLimiter{stats: make(map[string]*ipStats)}
	go rl.cleanupLoop()
	return rl
}

// allow 判断该 IP 是否允许通过，返回 (allowed, reason)。
// reason 在 allowed=false 时填，给前端看。
func (rl *rateLimiter) allow(ip string) (bool, string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	s, ok := rl.stats[ip]
	if !ok {
		s = &ipStats{}
		rl.stats[ip] = s
	}
	s.lastSeen = now

	// 1. 清理过期的一分钟窗口请求（滑动窗口）
	cutoff := now.Add(-time.Minute)
	i := 0
	for i < len(s.minuteWindow) && s.minuteWindow[i].Before(cutoff) {
		i++
	}
	s.minuteWindow = s.minuteWindow[i:]

	// 检查每分钟限制
	if len(s.minuteWindow) >= rateLimitPerMinute {
		return false, "too many requests, please slow down"
	}

	// 2. 检查每天免费限制（滚动 24 小时窗口）
	if s.dayResetAt.IsZero() || now.After(s.dayResetAt) {
		s.dayCount = 0
		s.dayResetAt = now.Add(24 * time.Hour)
	}
	if s.dayCount >= freeLimitPerDay {
		return false, "daily free limit reached"
	}

	// 通过，记录这次请求
	s.minuteWindow = append(s.minuteWindow, now)
	s.dayCount++
	return true, ""
}

// cleanupLoop 每 10 分钟清理一次超过 24 小时没活动的 IP，防内存泄漏。
func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-24 * time.Hour)
		for ip, s := range rl.stats {
			if s.lastSeen.Before(cutoff) {
				delete(rl.stats, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// clientIP 从请求里提取客户端 IP。
// Railway/Vercel 等反代会带 X-Forwarded-For，要取第一个（最原始的客户端 IP）。
// 不信任 X-Forwarded-For 是因为没做信任代理配置（MVP 阶段接受），
// 攻击者可以伪造这个 header 绕过限流，但 Railway 的边缘代理会覆盖真实值。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For: client, proxy1, proxy2 —— 取第一个
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

// rateLimitMiddleware 包装 handler，加 IP 限流。
// 只挂在 /generate 上（要花钱调 DeepSeek），/health 和 /create-checkout 不限流。
func rateLimitMiddleware(rl *rateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		allowed, reason := rl.allow(ip)
		if !allowed {
			writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: reason})
			return
		}
		next(w, r)
	}
}
