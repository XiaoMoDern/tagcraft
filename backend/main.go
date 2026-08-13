package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	loadEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8787"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/generate", handleGenerate)
	mux.HandleFunc("/create-checkout", handleCreateCheckout)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	handler := withCORS(mux)

	log.Printf("TagCraft server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

// loadEnv 从 .env 文件加载环境变量（不引依赖，手写解析）。
// 只在环境变量未设置时填充，系统环境变量优先。.env 不存在则跳过。
func loadEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
