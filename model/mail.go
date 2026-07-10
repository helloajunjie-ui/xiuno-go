package model

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"xiuno/core"
)

// SMTPConfig 引用 core 包中的定义
// 对应 PHP: reference/model/smtp.func.php

// Mailer 邮件发送器
type Mailer struct {
	Configs []core.SMTPConfig // 支持多个 SMTP 配置，按顺序尝试
}

// NewMailer 创建邮件发送器
func NewMailer(configs []core.SMTPConfig) *Mailer {
	return &Mailer{Configs: configs}
}

// Send 发送邮件，按配置列表顺序尝试，直到成功
func (m *Mailer) Send(to, subject, body string) error {
	if len(m.Configs) == 0 {
		return fmt.Errorf("mail: 未配置 SMTP 服务器")
	}
	var lastErr error
	for _, cfg := range m.Configs {
		err := m.sendWithConfig(cfg, to, subject, body)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("mail: 所有 SMTP 服务器均失败，最后错误: %w", lastErr)
}

func (m *Mailer) sendWithConfig(cfg core.SMTPConfig, to, subject, body string) error {
	if cfg.Host == "" || cfg.Port == 0 {
		return fmt.Errorf("mail: SMTP 配置不完整")
	}

	from := cfg.Email
	if from == "" {
		from = cfg.User
	}

	// 构建邮件头
	fromName := cfg.FromName
	if fromName == "" {
		fromName = from
	}
	header := make(map[string]string)
	header["From"] = fmt.Sprintf("%s <%s>", encodeRFC1123(fromName), from)
	header["To"] = to
	header["Subject"] = encodeRFC1123(subject)
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=UTF-8"
	header["Content-Transfer-Encoding"] = "base64"
	header["Date"] = time.Now().Format(time.RFC1123Z)

	// 构建完整消息
	var msg strings.Builder
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(base64Encode([]byte(body)))

	addr := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))

	switch cfg.Secure {
	case "tls":
		return m.sendTLS(cfg, addr, from, to, msg.String())
	case "starttls":
		return m.sendSTARTTLS(cfg, addr, from, to, msg.String())
	default:
		return m.sendPlain(cfg, addr, from, to, msg.String())
	}
}

// sendPlain 明文 SMTP（端口 25）
func (m *Mailer) sendPlain(cfg core.SMTPConfig, addr, from, to, msg string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("mail: 连接失败 %s: %w", addr, err)
	}
	defer client.Close()

	if err = client.Hello("localhost"); err != nil {
		return err
	}

	// 检查是否支持 STARTTLS，如果支持则升级
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: cfg.Host, InsecureSkipVerify: false}
		if err = client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("mail: STARTTLS 升级失败: %w", err)
		}
	}

	return m.authAndSend(client, cfg, from, to, msg)
}

// sendSTARTTLS 显式 TLS（端口 587）
func (m *Mailer) sendSTARTTLS(cfg core.SMTPConfig, addr, from, to, msg string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("mail: 连接失败 %s: %w", addr, err)
	}
	defer client.Close()

	tlsCfg := &tls.Config{ServerName: cfg.Host, InsecureSkipVerify: false}
	if err = client.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("mail: STARTTLS 失败: %w", err)
	}

	return m.authAndSend(client, cfg, from, to, msg)
}

// sendTLS 隐式 TLS（端口 465）
func (m *Mailer) sendTLS(cfg core.SMTPConfig, addr, from, to, msg string) error {
	tlsCfg := &tls.Config{ServerName: cfg.Host, InsecureSkipVerify: false}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("mail: TLS 连接失败 %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	return m.authAndSend(client, cfg, from, to, msg)
}

func (m *Mailer) authAndSend(client *smtp.Client, cfg core.SMTPConfig, from, to, msg string) error {
	// 认证
	if cfg.User != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mail: 认证失败: %w", err)
		}
	}

	// 发件人
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail: MAIL FROM 失败: %w", err)
	}

	// 收件人
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mail: RCPT TO 失败: %w", err)
	}

	// 发送数据
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: DATA 失败: %w", err)
	}
	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	return w.Close()
}

// encodeRFC1123 对邮件头中的非 ASCII 文本进行 MIME 编码
func encodeRFC1123(s string) string {
	// 检查是否包含非 ASCII 字符
	for _, r := range s {
		if r > 127 {
			return fmt.Sprintf("=?UTF-8?B?%s?=", base64Encode([]byte(s)))
		}
	}
	return s
}

// base64Encode Base64 编码（RFC 2045 兼容：每 76 字符插入 \r\n 换行）
func base64Encode(data []byte) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	const lineLen = 76
	var result strings.Builder
	col := 0
	for i := 0; i < len(data); i += 3 {
		var buf [3]byte
		var n int
		for j := 0; j < 3 && i+j < len(data); j++ {
			buf[j] = data[i+j]
			n++
		}
		// 第一组 6 位
		result.WriteByte(charset[buf[0]>>2])
		col++
		// 第二组 6 位
		result.WriteByte(charset[(buf[0]&0x03)<<4|buf[1]>>4])
		col++
		if n < 2 {
			result.WriteByte('=')
			result.WriteByte('=')
			col += 2
		} else {
			result.WriteByte(charset[(buf[1]&0x0f)<<2|buf[2]>>6])
			col++
			if n < 3 {
				result.WriteByte('=')
				col++
			} else {
				result.WriteByte(charset[buf[2]&0x3f])
				col++
			}
		}
		// RFC 2045: 每 76 字符插入 CRLF
		if col >= lineLen {
			result.WriteString("\r\n")
			col = 0
		}
	}
	return result.String()
}

// DefaultSMTPConfigs 返回默认 SMTP 配置（从 xiuno.json 加载后覆盖）
func DefaultSMTPConfigs() []core.SMTPConfig {
	return []core.SMTPConfig{}
}

// ValidateSMTPConfig 校验 SMTP 配置是否可用（尝试连接但不发邮件）
func ValidateSMTPConfig(cfg core.SMTPConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("SMTP 服务器地址不能为空")
	}
	if cfg.Port == 0 {
		return fmt.Errorf("SMTP 端口不能为空")
	}
	if cfg.User == "" {
		return fmt.Errorf("SMTP 用户名不能为空")
	}
	if cfg.Pass == "" {
		return fmt.Errorf("SMTP 密码不能为空")
	}

	// 尝试 TCP 连接
	addr := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("SMTP 服务器连接失败: %w", err)
	}
	conn.Close()
	return nil
}

// SmtpCount 返回 SMTP 配置数量
// 对应 PHP: smtp_count()
func SmtpCount(configs []core.SMTPConfig) int {
	return len(configs)
}

// SmtpMaxID 返回最大 SMTP 配置索引
// 对应 PHP: smtp_maxid()
func SmtpMaxID(configs []core.SMTPConfig) int {
	return len(configs) - 1
}
