// 本文件连接账号服务、邮件令牌存储和邮件发送服务，编排重置密码与接受邀请。
// 令牌消费、密码保存和会话处理分别调用各自入口，这些步骤不共享此处创建的事务。
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/email"
)

// EmailTokenService 负责签发邮件链接并处理用户提交的令牌和密码。
// 账号查找与密码摘要交给 UserService，持久化刷新会话撤销交给 AuthService，
// 邮件正文和传输交给 email.Service；构造函数本身不检查这些依赖是否为空。
type EmailTokenService struct {
	tokens    *EmailTokenStore
	users     *UserService
	auth      *AuthService
	mailer    *email.Service
	publicURL string
	log       *slog.Logger

	// 等待组跟踪异步重置邮件，Close 等待已登记的发送结束。
	// 此字段不阻止调用方在关闭期间继续登记发送，生命周期由调用方协调。
	wg sync.WaitGroup
}

// NewEmailTokenService 保存业务依赖，并从公开地址尾部移除一个斜杠。
// 地址的协议、主机和路径不在此验证，后续直接与邮件入口路径拼接。
func NewEmailTokenService(tokens *EmailTokenStore, users *UserService, authService *AuthService, mailer *email.Service, publicURL string) *EmailTokenService {
	return &EmailTokenService{
		tokens:    tokens,
		users:     users,
		auth:      authService,
		mailer:    mailer,
		publicURL: strings.TrimSuffix(publicURL, "/"),
		log:       slog.Default().With("component", "EmailTokenService"),
	}
}

// RequestPasswordReset 按用户名或邮箱找账号，找到后签发一小时有效的密码重置令牌，
// 并把链接发送到账号邮箱。账号查找的所有错误均返回 nil，签发和发送失败返回错误。
// 令牌在发送前已经保存，发送失败不会在此删除令牌；上下文只传给邮件发送入口。
func (s *EmailTokenService) RequestPasswordReset(ctx context.Context, identifier string) error {
	// 查找失败不向调用方暴露账号是否存在，也不继续生成链接。
	user, err := s.users.GetUserByIdentifier(identifier)
	if err != nil {
		return nil
	}

	rawToken, err := s.tokens.Issue(user.ID, PurposePasswordReset, PasswordResetTokenTTL)
	if err != nil {
		return fmt.Errorf("issue password reset token: %w", err)
	}

	// 数据库签发成功后才尝试发送，邮件失败的错误保留原始原因供上层记录。
	link := s.buildLink("/reset-password", rawToken)
	if err := s.mailer.SendPasswordResetEmail(ctx, user.Email, link); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}
	return nil
}

// RequestPasswordResetAsync 登记任务后启动后台协程，不等待账号查询和邮件发送。
// 它使用后台上下文，不受原 HTTP 请求取消影响；发送错误记录为警告，无法通过
// 该无返回值入口交还给调用方。任务退出时递减等待组，供 Close 等待。
func (s *EmailTokenService) RequestPasswordResetAsync(identifier string) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.RequestPasswordReset(context.Background(), identifier); err != nil {
			s.log.Warn("failed to send password reset email", "error", err)
		}
	}()
}

// Close 等待已登记的异步重置邮件结束，不取消任务，也不关闭令牌存储和邮件服务。
func (s *EmailTokenService) Close() {
	s.wg.Wait()
}

// ConfirmPasswordReset 按重置用途验证并消费令牌，再保存新密码，尝试撤销账号的
// 持久化刷新会话，返回重新查询的账号。它不签发会话，也不直接使已签发的访问 JWT 失效。
// 消费后密码写入失败不会恢复令牌；撤销会话失败只记警告，仍继续读取账号。
func (s *EmailTokenService) ConfirmPasswordReset(rawToken, newPassword string) (*User, error) {
	tok, err := s.tokens.Resolve(rawToken, PurposePasswordReset)
	if err != nil {
		return nil, err
	}
	// 先占用一次性令牌，防止两个请求都继续修改密码。后续失败不回滚此状态。
	if err := s.tokens.Consume(tok.ID); err != nil {
		return nil, err
	}

	if err := s.users.UpdatePassword(tok.UserID, newPassword); err != nil {
		return nil, err
	}
	// 密码已写入后才撤销刷新会话，失败只记录日志，不覆盖后续账号查询结果。
	if err := s.auth.RevokeAllUserSessions(tok.UserID); err != nil {
		s.log.Warn("failed to revoke sessions after password reset", "userID", tok.UserID, "error", err)
	}

	return s.users.GetUserByID(tok.UserID)
}

// IssueInvite 使用传入账号的标识签发七天有效的邀请令牌，再发送到该对象的邮箱。
// 此入口不创建或重新查找账号；发送失败时已保存的令牌仍然保留。
func (s *EmailTokenService) IssueInvite(ctx context.Context, user *User) error {
	rawToken, err := s.tokens.Issue(user.ID, PurposeInvite, InviteTokenTTL)
	if err != nil {
		return fmt.Errorf("issue invite token: %w", err)
	}

	link := s.buildLink("/accept-invite", rawToken)
	return s.mailer.SendInviteEmail(ctx, user.Email, link)
}

// ConfirmInvite 验证邀请用途并消费令牌，通过账号服务保存密码、清除待设密码标记，
// 再返回账号。它不撤销已有会话，也不在此自动签发登录会话。
// 各步骤依次执行，消费后发生的账号写入或查询失败不会让该令牌恢复可用。
func (s *EmailTokenService) ConfirmInvite(rawToken, newPassword string) (*User, error) {
	tok, err := s.tokens.Resolve(rawToken, PurposeInvite)
	if err != nil {
		return nil, err
	}
	if err := s.tokens.Consume(tok.ID); err != nil {
		return nil, err
	}

	if err := s.users.CompleteInvite(tok.UserID, newPassword); err != nil {
		return nil, err
	}

	return s.users.GetUserByID(tok.UserID)
}

// buildLink 直接拼接公开地址、入口路径和 token 查询参数，不执行 URL 编码。
// 当前签发格式由标识、点和十六进制密钥组成，调用方传入固定的业务入口路径。
func (s *EmailTokenService) buildLink(path, rawToken string) string {
	return fmt.Sprintf("%s%s?token=%s", s.publicURL, path, rawToken)
}
