# 链接、图片与表格

## 指向页面与外部资料

标准链接由显示文字和目标地址组成。目标必须是实际存在的页面或资源。

```markdown
[入门教程](/markdown-getting-started)
[CommonMark 规范](https://spec.commonmark.org/)
```

站内地址应使用实例当前路径规则。在带基础路径的实例中，检查实际点击结果，不要把另一个实例的完整地址照搬过来。页面移动后也应检查入链，而不是只检查新页面能否打开。

## 图片和附件

上传资源后使用系统返回的地址。下面使用程序自带图片，预览中应出现 LeafDoc 图标。

![LeafDoc 图标](/defaults/leafdoc-icon-180.png)

```markdown
![图片说明](/defaults/leafdoc-icon-180.png)
```

图片说明用于表达图片含义。需要下载的文件使用普通链接。不要把本机磁盘路径写成其他读者的附件地址。文件是否可读仍受到实例访问设置和接口权限约束。

## 表格

```markdown
| 检查项 | 结果 |
| --- | --- |
| 首页 | 可以打开 |
| 登录 | 进入文档树 |
```

表格适合比较并列信息，长篇操作步骤应写成列表。单元格包含竖线时需要转义，否则会形成新的列。列数不同导致显示错位时，先检查分隔行和每一行的竖线。

## 链接验证

保存后逐个点击重要链接。图片未显示时核对资源是否上传、地址是否正确、当前账号是否具有读取权限。

## 教程导航

- [Markdown 入门与完整写作流程](/markdown-getting-started)
- [标题、文本格式与代码](/markdown-text-and-code)
- [列表、引用与内容组织](/markdown-lists-and-structure)
- [链接、图片与表格](/markdown-links-images-and-tables)
- [扩展语法、实际示例与排错](/markdown-extensions-and-troubleshooting)
