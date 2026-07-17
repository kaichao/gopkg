package security

import (
	"context"
	"net/http"
	"time"
)

// AuditRecorder 记录 HTTP 请求的审计日志。
// 实现为 http.Handler 中间件，拦截写操作自动记录。
type AuditRecorder struct {
	store AuditStore
}

// NewAuditRecorder 创建审计记录器。
func NewAuditRecorder(store AuditStore) *AuditRecorder {
	return &AuditRecorder{store: store}
}

// Middleware 返回审计中间件。
// 自动记录 POST/PUT/DELETE 操作。GET/HEAD 请求不记录。
func (recorder *AuditRecorder) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter 捕获状态码
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, req)

		// 仅记录写操作
		if req.Method == "GET" || req.Method == "HEAD" || req.Method == "OPTIONS" {
			return
		}

		userID := "anonymous"
		if id := IdentityFromContext(req.Context()); id != nil {
			userID = id.Subject()
		}

		detail := req.Method + " " + req.URL.Path
		if rec.status >= 400 {
			detail += " (error)"
		}

		// 异步写入，不阻塞响应
		go recorder.store.Record(context.Background(), AuditEntry{
			UserID:    userID,
			Action:    req.Method,
			Resource:  req.URL.Path,
			Detail:    detail,
			Timestamp: start,
		})
	})
}

// Record 手动记录一条审计日志（供业务代码调用）。
func Record(ctx context.Context, store AuditStore, userID, action, resource, detail string) {
	if store == nil {
		return
	}
	go func() {
		_ = store.Record(context.Background(), AuditEntry{
			UserID:    userID,
			Action:    action,
			Resource:  resource,
			Detail:    detail,
			Timestamp: time.Now(),
		})
	}()
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
