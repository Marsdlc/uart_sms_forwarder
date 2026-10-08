package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dushixiang/uart_sms_forwarder/internal/service"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

func TestSMSFilterAPI(t *testing.T) {
	// nil 服务和通知器保证草稿测试与非法配置校验不会读写数据库或发送通知。
	handler := NewPropertyHandler(zap.NewNop(), nil, nil)
	e := echo.New()
	e.POST("/api/sms-filter/test", handler.TestSMSFilter)
	e.PUT("/api/properties/:id", handler.SetProperty)

	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		status  int
		matched bool
		forward bool
	}{
		{name: "include match", body: `{"config":{"enabled":true,"mode":"include","pattern":"验证码"},"content":"验证码123456"}`, status: http.StatusOK, matched: true, forward: true},
		{name: "exclude match", body: `{"config":{"enabled":true,"mode":"exclude","pattern":"优惠"},"content":"优惠活动"}`, status: http.StatusOK, matched: true},
		{name: "disabled", body: `{"config":{"enabled":false,"mode":"include","pattern":""},"content":"任意短信"}`, status: http.StatusOK, forward: true},
		{name: "invalid expression", body: `{"config":{"enabled":true,"mode":"include","pattern":"["}}`, status: http.StatusBadRequest},
		{name: "missing config", body: `{}`, status: http.StatusBadRequest},
		{name: "null config", body: `{"config":null}`, status: http.StatusBadRequest},
		{name: "malformed body", body: `{`, status: http.StatusBadRequest},
		{name: "invalid field type", body: `{"config":{"enabled":"yes","mode":"include"}}`, status: http.StatusBadRequest},
		{name: "reject invalid save", method: http.MethodPut, path: "/api/properties/sms_filter_config", body: `{"value":{"enabled":true,"mode":"include","pattern":"["}}`, status: http.StatusBadRequest},
		{name: "reject null save", method: http.MethodPut, path: "/api/properties/sms_filter_config", body: `{"value":null}`, status: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, path := tt.method, tt.path
			if method == "" {
				method = http.MethodPost
			}
			if path == "" {
				path = "/api/sms-filter/test"
			}
			request := httptest.NewRequest(method, path, strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			e.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.status, response.Body.String())
			}
			if tt.status == http.StatusOK {
				var result service.SMSFilterResult
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Matched != tt.matched || result.Forward != tt.forward {
					t.Fatalf("result = %+v, want matched=%v forward=%v", result, tt.matched, tt.forward)
				}
			}
		})
	}
}
