// 本文件保存账号通知邮件的五种语言文本，内容直接交给邮件生成代码使用。
package localization

// MailText 分别保存邮件主题、开场文字、操作引导和有效期提示。
// 这些字段仅是文本，不负责生成链接，也不决定令牌的真实失效时间。
type MailText struct{ Subject, Intro, Action, Expiry string }

// 每种语言的第零项用于密码重置，第一项用于账号邀请，顺序由 Mail 按索引选择。
var mail = map[string][]MailText{
	"zh": {{"重置 LeafDoc 密码", "您好，", "请打开以下链接重置密码：", "此链接将在 1 小时后失效。如果您没有申请重置密码，请忽略此邮件。"}, {"LeafDoc 账户邀请", "您好，", "您已受邀加入 LeafDoc，请打开以下链接设置密码：", "此链接将在 7 天后失效。"}},
	"en": {{"Reset your LeafDoc password", "Hello,", "Open this link to reset your password:", "This link expires in 1 hour. Ignore this email if you did not request a reset."}, {"You've been invited to LeafDoc", "Hello,", "Open this link to set your password:", "This link expires in 7 days."}},
	"de": {{"LeafDoc-Passwort zurücksetzen", "Hallo,", "Öffnen Sie diesen Link, um Ihr Passwort zurückzusetzen:", "Dieser Link läuft in 1 Stunde ab. Ohne eigene Anfrage können Sie diese E-Mail ignorieren."}, {"Einladung zu LeafDoc", "Hallo,", "Öffnen Sie diesen Link, um Ihr Passwort festzulegen:", "Dieser Link läuft in 7 Tagen ab."}},
	"es": {{"Restablecer la contraseña de LeafDoc", "Hola,", "Abra este enlace para restablecer su contraseña:", "Este enlace caduca en 1 hora. Ignore este correo si no solicitó el cambio."}, {"Invitación a LeafDoc", "Hola,", "Abra este enlace para establecer su contraseña:", "Este enlace caduca en 7 días."}},
	"fr": {{"Réinitialiser votre mot de passe LeafDoc", "Bonjour,", "Ouvrez ce lien pour réinitialiser votre mot de passe :", "Ce lien expire dans 1 heure. Ignorez ce message si vous n'avez pas fait cette demande."}, {"Invitation à LeafDoc", "Bonjour,", "Ouvrez ce lien pour définir votre mot de passe :", "Ce lien expire dans 7 jours."}},
}

// Mail 在调用时读取语言，语言不存在或对应列表为空时回退到中文。
// invite 为 true 返回邀请文本，否则返回密码重置文本；返回结构是值副本。
func Mail(invite bool) MailText {
	values := mail[Language()]
	if len(values) == 0 {
		values = mail["zh"]
	}
	if invite {
		return values[1]
	}
	return values[0]
}
