package localization

// MailText contains the subject and plain paragraphs for an account notification.
type MailText struct{ Subject, Intro, Action, Expiry string }

var mail = map[string][]MailText{
	"zh": {{"重置 LeafWiki 密码", "您好，", "请打开以下链接重置密码：", "此链接将在 1 小时后失效。如果您没有申请重置密码，请忽略此邮件。"}, {"LeafWiki 账户邀请", "您好，", "您已受邀加入 LeafWiki，请打开以下链接设置密码：", "此链接将在 7 天后失效。"}},
	"en": {{"Reset your LeafWiki password", "Hello,", "Open this link to reset your password:", "This link expires in 1 hour. Ignore this email if you did not request a reset."}, {"You've been invited to LeafWiki", "Hello,", "Open this link to set your password:", "This link expires in 7 days."}},
	"de": {{"LeafWiki-Passwort zurücksetzen", "Hallo,", "Öffnen Sie diesen Link, um Ihr Passwort zurückzusetzen:", "Dieser Link läuft in 1 Stunde ab. Ohne eigene Anfrage können Sie diese E-Mail ignorieren."}, {"Einladung zu LeafWiki", "Hallo,", "Öffnen Sie diesen Link, um Ihr Passwort festzulegen:", "Dieser Link läuft in 7 Tagen ab."}},
	"es": {{"Restablecer la contraseña de LeafWiki", "Hola,", "Abra este enlace para restablecer su contraseña:", "Este enlace caduca en 1 hora. Ignore este correo si no solicitó el cambio."}, {"Invitación a LeafWiki", "Hola,", "Abra este enlace para establecer su contraseña:", "Este enlace caduca en 7 días."}},
	"fr": {{"Réinitialiser votre mot de passe LeafWiki", "Bonjour,", "Ouvrez ce lien pour réinitialiser votre mot de passe :", "Ce lien expire dans 1 heure. Ignorez ce message si vous n'avez pas fait cette demande."}, {"Invitation à LeafWiki", "Bonjour,", "Ouvrez ce lien pour définir votre mot de passe :", "Ce lien expire dans 7 jours."}},
}

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
