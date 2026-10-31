// 将状态更新排入微任务，在当前同步调用栈结束后执行，不创建计时器。
// 函数不返回调度产生的 Promise，也不提供取消方式，回调抛错会使该 Promise 拒绝。
export function deferStateUpdate(fn: () => void) {
  Promise.resolve().then(fn)
}
