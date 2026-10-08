package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/dushixiang/uart_sms_forwarder/internal/models"
	"gorm.io/gorm"
)

const (
	PropertyIDSMSFilterConfig = "sms_filter_config"
	SMSFilterModeInclude      = "include"
	SMSFilterModeExclude      = "exclude"
)

// SMSFilterResult 同时供实际转发和规则测试使用。
type SMSFilterResult struct {
	Matched bool `json:"matched"`
	Forward bool `json:"forward"`
}

func DefaultSMSFilterConfig() models.SMSFilterConfig {
	return models.SMSFilterConfig{Mode: SMSFilterModeInclude}
}

func compileSMSFilter(config models.SMSFilterConfig) (*regexp.Regexp, error) {
	if config.Mode != SMSFilterModeInclude && config.Mode != SMSFilterModeExclude {
		return nil, fmt.Errorf("过滤模式必须为 include 或 exclude")
	}
	if strings.TrimSpace(config.Pattern) == "" {
		if config.Enabled {
			return nil, fmt.Errorf("启用短信过滤时必须填写正则表达式")
		}
		return nil, nil
	}

	// 保留用户填写的空格；空格也可能是规则的一部分。
	expression, err := regexp.Compile(config.Pattern)
	if err != nil {
		return nil, fmt.Errorf("正则表达式无效: %w", err)
	}
	return expression, nil
}

func ValidateSMSFilterConfig(config models.SMSFilterConfig) error {
	_, err := compileSMSFilter(config)
	return err
}

// EvaluateSMSFilter 匹配原始正文，不匹配通知附加的发送号码和时间。
// 关闭过滤时始终放行；无效规则返回错误，由调用方决定如何处理。
func EvaluateSMSFilter(config models.SMSFilterConfig, content string) (SMSFilterResult, error) {
	expression, err := compileSMSFilter(config)
	if err != nil {
		return SMSFilterResult{}, err
	}
	if !config.Enabled {
		return SMSFilterResult{Forward: true}, nil
	}

	matched := expression.MatchString(content)
	forward := matched
	if config.Mode == SMSFilterModeExclude {
		forward = !matched
	}
	return SMSFilterResult{Matched: matched, Forward: forward}, nil
}

func (s *PropertyService) GetSMSFilterConfig(ctx context.Context) (models.SMSFilterConfig, error) {
	config := DefaultSMSFilterConfig()
	if err := s.GetValue(ctx, PropertyIDSMSFilterConfig, &config); err != nil {
		// 老数据库尚未初始化该配置时，保留默认关闭的行为。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DefaultSMSFilterConfig(), nil
		}
		return models.SMSFilterConfig{}, fmt.Errorf("获取短信过滤配置失败: %w", err)
	}
	return config, nil
}

func (s *PropertyService) SetSMSFilterConfig(ctx context.Context, config models.SMSFilterConfig) error {
	if err := ValidateSMSFilterConfig(config); err != nil {
		return err
	}
	return s.Set(ctx, PropertyIDSMSFilterConfig, "短信过滤配置", config)
}
