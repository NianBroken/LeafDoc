// 页面树、搜索与导入响应共用节点类别字符串，常量值直接对应接口传输值。
export const NODE_KIND_PAGE = 'page'
// section 表示章节类节点，具体内容与操作权限由相应业务模块解释。
export const NODE_KIND_SECTION = 'section'
// PageKind 只约束 TypeScript 调用，不在运行时验证服务器返回数据。
export type PageKind = typeof NODE_KIND_PAGE | typeof NODE_KIND_SECTION
