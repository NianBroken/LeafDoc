package http

// 本文件只在测试构建中将两个内部 HTML 助手暴露给外部 http_test 包。
// 变量直接引用实际函数，不复制实现，外部测试因此验证生产代码使用的同一处理逻辑。

var BuildCustomStylesheetTag = buildCustomStylesheetTag
var InjectIntoHead = injectIntoHead
