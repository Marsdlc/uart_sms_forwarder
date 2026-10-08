package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dushixiang/uart_sms_forwarder/config"
	"github.com/dushixiang/uart_sms_forwarder/internal/models"
	"github.com/go-orz/cache"
	"go.uber.org/zap"
)

func TestEvaluateSMSFilter(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		mode    string
		pattern string
		content string
		matched bool
		forward bool
		wantErr bool
	}{
		{name: "disabled empty rule", mode: SMSFilterModeInclude, forward: true},
		{name: "disabled exclude rule", mode: SMSFilterModeExclude, pattern: "验证码", content: "验证码", forward: true},
		{name: "include match", enabled: true, mode: SMSFilterModeInclude, pattern: "验证码|校验码", content: "您的校验码是123456", matched: true, forward: true},
		{name: "include no match", enabled: true, mode: SMSFilterModeInclude, pattern: "验证码", content: "优惠活动"},
		{name: "exclude match", enabled: true, mode: SMSFilterModeExclude, pattern: "优惠|退订", content: "优惠活动", matched: true},
		{name: "exclude no match", enabled: true, mode: SMSFilterModeExclude, pattern: "优惠|退订", content: "验证码123456", forward: true},
		{name: "empty content", enabled: true, mode: SMSFilterModeInclude, pattern: "验证码"},
		{name: "anchor matches empty content", enabled: true, mode: SMSFilterModeInclude, pattern: "^$", matched: true, forward: true},
		{name: "case insensitive", enabled: true, mode: SMSFilterModeInclude, pattern: "(?i)otp", content: "Your OTP is 123456", matched: true, forward: true},
		{name: "multiline body", enabled: true, mode: SMSFilterModeInclude, pattern: "(?s)验证码.*有效", content: "验证码123456\n五分钟内有效", matched: true, forward: true},
		{name: "preserve spaces", enabled: true, mode: SMSFilterModeInclude, pattern: " 验证码 ", content: "验证码123456"},
		{name: "invalid pattern", enabled: true, mode: SMSFilterModeInclude, pattern: "[", wantErr: true},
		{name: "invalid pattern while disabled", mode: SMSFilterModeInclude, pattern: "[", wantErr: true},
		{name: "empty enabled pattern", enabled: true, mode: SMSFilterModeInclude, wantErr: true},
		{name: "blank enabled pattern", enabled: true, mode: SMSFilterModeInclude, pattern: " \n\t", wantErr: true},
		{name: "unknown mode", mode: "invalid", wantErr: true},
		{name: "unsupported lookahead", enabled: true, mode: SMSFilterModeInclude, pattern: "验证码(?=123)", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := models.SMSFilterConfig{Enabled: tt.enabled, Mode: tt.mode, Pattern: tt.pattern}
			result, err := EvaluateSMSFilter(rule, tt.content)
			if (err != nil) != tt.wantErr {
				t.Fatalf("EvaluateSMSFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && (result.Matched != tt.matched || result.Forward != tt.forward) {
				t.Fatalf("EvaluateSMSFilter() = %+v, want matched=%v forward=%v", result, tt.matched, tt.forward)
			}
			if err := ValidateSMSFilterConfig(rule); (err != nil) != tt.wantErr {
				t.Fatalf("ValidateSMSFilterConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 使用本地 Webhook 验证实际通知路径，以及系统通知不会被正文过滤。
func TestSMSFilterNotificationRouting(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	logger := zap.NewNop()
	properties := &PropertyService{logger: logger, cache: cache.New[string, *models.Property](time.Minute)}
	setProperty := func(id string, value interface{}) {
		t.Helper()
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		properties.cache.Set(id, &models.Property{ID: id, Value: string(encoded)}, time.Hour)
	}
	setProperty(PropertyIDNotificationChannels, []models.NotificationChannelConfig{
		{Type: "webhook", Enabled: true, Config: map[string]interface{}{
			"url": server.URL, "body": `{"content":"{{content}}"}`, "contentType": "application/json",
		}},
	})
	setProperty(PropertyIDSMSFilterConfig, models.SMSFilterConfig{
		Enabled: true, Mode: SMSFilterModeInclude, Pattern: "验证码",
	})
	serial := NewSerialService(logger, config.SerialConfig{}, nil, NewNotifier(logger), properties)
	ctx := context.Background()

	// 发送方匹配规则不会放行正文不匹配的短信。
	serial.sendNotification(ctx, IncomingSMS{From: "验证码", Content: "优惠活动"})
	if got := requests.Load(); got != 0 {
		t.Fatalf("filtered SMS sent %d requests", got)
	}
	serial.sendNotification(ctx, IncomingSMS{From: "10086", Content: "验证码123456"})
	if got := requests.Load(); got != 1 {
		t.Fatalf("matching SMS sent %d requests, want 1", got)
	}
	for _, kind := range []string{"sms", "call", "flymode"} {
		serial.sendNotificationMessage(ctx, NotificationMessage{Type: kind, From: "系统", Content: "状态通知"})
	}
	if got := requests.Load(); got != 4 {
		t.Fatalf("system notifications sent %d requests, want 4 total", got)
	}

	setProperty(PropertyIDSMSFilterConfig, models.SMSFilterConfig{
		Enabled: true, Mode: SMSFilterModeInclude, Pattern: "[",
	})
	serial.sendNotification(ctx, IncomingSMS{Content: "验证码123456"})
	properties.cache.Set(PropertyIDSMSFilterConfig, &models.Property{Value: "invalid JSON"}, time.Hour)
	serial.sendNotification(ctx, IncomingSMS{Content: "验证码123456"})
	if got := requests.Load(); got != 4 {
		t.Fatalf("invalid configs must skip forwarding, got %d requests", got)
	}

	setProperty(PropertyIDSMSFilterConfig, DefaultSMSFilterConfig())
	serial.sendNotification(ctx, IncomingSMS{Content: "优惠活动"})
	if got := requests.Load(); got != 5 {
		t.Fatalf("disabled filter sent %d requests, want 5 total", got)
	}
}
