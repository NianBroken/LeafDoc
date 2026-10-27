package localization

// Welcome supplies the initial example document in the instance language.
func Welcome() (string, string) {
	text := map[string][2]string{
		"zh": {"欢迎使用 LeafWiki", "# 欢迎使用 LeafWiki\n\nLeafWiki 是一个轻量的在线文档系统，文档以 Markdown 文件保存在本机。\n\n## 主要功能\n\n- 使用页面和分区组织文档\n- 全文搜索与附件管理\n- 文档历史、快照备份与恢复\n- 管理员、编辑者和只读用户权限\n- 支持访客只读访问\n\n## 开始使用\n\n1. 登录后使用侧栏的加号创建页面。\n2. 使用 Markdown 编写标题、列表、代码和链接。\n3. 使用分区整理相关页面。\n4. 将文件拖入编辑器上传附件。\n\n管理员可以在全局设置中统一调整语言、日期、时间、自动保存、目录和公开访问。\n"},
		"en": {"Welcome to LeafWiki", "# Welcome to LeafWiki\n\nLeafWiki stores documents as Markdown files on disk.\n\n## Features\n\n- Hierarchical pages and sections\n- Full-text search and attachments\n- Revision history, snapshots and restore\n- Administrator, editor and viewer roles\n- Public read-only access\n\n## Getting started\n\n1. Sign in and create a page from the sidebar.\n2. Write headings, lists, code and links in Markdown.\n3. Organize pages in sections.\n4. Drag files into the editor to upload attachments.\n"},
		"de": {"Willkommen bei LeafWiki", "# Willkommen bei LeafWiki\n\nLeafWiki speichert Dokumente als Markdown-Dateien.\n\n## Funktionen\n\n- Seiten und Bereiche\n- Volltextsuche und Anhänge\n- Versionsverlauf und Sicherungen\n- Rollen und öffentlicher Lesezugriff\n\n## Erste Schritte\n\nMelden Sie sich an und erstellen Sie eine Seite über die Seitenleiste. Schreiben Sie Texte in Markdown und ziehen Sie Dateien in den Editor.\n"},
		"es": {"Bienvenido a LeafWiki", "# Bienvenido a LeafWiki\n\nLeafWiki guarda documentos como archivos Markdown.\n\n## Funciones\n\n- Páginas y secciones\n- Búsqueda de texto y archivos adjuntos\n- Historial y copias de seguridad\n- Roles y acceso público de lectura\n\n## Primeros pasos\n\nInicie sesión y cree una página desde la barra lateral. Escriba en Markdown y arrastre archivos al editor.\n"},
		"fr": {"Bienvenue dans LeafWiki", "# Bienvenue dans LeafWiki\n\nLeafWiki conserve les documents sous forme de fichiers Markdown.\n\n## Fonctionnalités\n\n- Pages et sections\n- Recherche et pièces jointes\n- Historique et sauvegardes\n- Rôles et accès public en lecture\n\n## Premiers pas\n\nConnectez-vous et créez une page depuis la barre latérale. Rédigez en Markdown et glissez des fichiers dans l’éditeur.\n"},
	}
	value, ok := text[Language()]
	if !ok {
		value = text["zh"]
	}
	return value[0], value[1]
}
