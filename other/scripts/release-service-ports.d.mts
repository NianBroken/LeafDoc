// 此声明描述相邻模块的同步入口，服务名数组交给平台实现解析为固定端口。
// 成功没有返回值，解析、启动或停止失败通过抛出的错误传递，不返回异步 Promise。
export function releaseFixedServicePorts(services: string[]): void;
