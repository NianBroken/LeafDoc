// 应用装配层提供认证模式读取函数，接口层无需直接导入配置状态模块。
// 每次读取调用当前函数，不在此复制或缓存返回对象。
export type AuthRuntimeConfig = {
  httpRemoteUserEnabled: boolean
  configLoadSucceeded: boolean
}
// 配置提供者尚未安装时按未启用远程用户且配置未成功加载处理。
let readConfig: () => AuthRuntimeConfig = () => ({
  httpRemoteUserEnabled: false,
  configLoadSucceeded: false,
})
// 包装对象保持稳定引用，内部读取函数可由配置入口替换。
export const authRuntime = { readConfig: () => readConfig() }
// 直接替换提供函数，不主动读取、验证或触发网络请求。
export function configureAuthRuntime(reader: () => AuthRuntimeConfig): void {
  readConfig = reader
}
