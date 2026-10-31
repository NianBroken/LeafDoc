// 用浏览器计时器返回延迟完成的 Promise，不阻塞当前线程。
// 实际触发时间遵循浏览器调度，不提供取消句柄，也不校验延迟参数。
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}
