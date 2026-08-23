package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// ContactCodeSender 向手机号/邮箱投递验证码。
// 默认实现 LoggingContactSender 仅记录日志，不接入真实短信/邮件网关；
// 接入真实网关时提供新实现并替换 wire provider 即可。
type ContactCodeSender interface {
	Send(ctx context.Context, dest, code string) error
}

// LoggingContactSender 将验证码写入日志的占位实现。
type LoggingContactSender struct {
	log *log.Helper
}

func NewLoggingContactSender(ctx *bootstrap.Context) *LoggingContactSender {
	return &LoggingContactSender{
		log: ctx.NewLoggerHelper("contact-sender/logging/core-service"),
	}
}

func (s *LoggingContactSender) Send(_ context.Context, dest, code string) error {
	s.log.Infof("contact verification code (logging sender): dest=%s code=%s", dest, code)
	return nil
}
