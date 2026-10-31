// 本文件为每次主动导航附加访问标识，供同一路径的多次访问区分状态。
// 只依赖路由位置所需的三个字段，不绑定完整的路由库对象类型。
type NavigationVisitLocation = {
  key?: string
  pathname: string
  state: unknown
}

// 写入和读取访问标识共用此状态键，避免字段名分散在调用处。
const NAVIGATION_VISIT_ID_KEY = 'leafdocVisitId'

// 仅检查非 null 对象，数组和其他对象也满足该判断，不要求普通对象原型。
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

// 浏览器提供 randomUUID 时使用它，否则组合当前毫秒和伪随机文本。
// 回退结果用于访问区分，不作为认证令牌，也不保证全局唯一。
function createVisitId(): string {
  if (
    typeof crypto !== 'undefined' &&
    typeof crypto.randomUUID === 'function'
  ) {
    return crypto.randomUUID()
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

// 对对象状态作浅拷贝，再覆盖访问标识，非对象输入从空状态开始。
// 不改变原对象，嵌套引用保留共享关系。
export function createNavigationVisitState(state?: unknown) {
  const nextState = isRecord(state) ? { ...state } : {}
  nextState[NAVIGATION_VISIT_ID_KEY] = createVisitId()
  return nextState
}

// 优先读取状态中的非空字符串标识，缺失时依次使用路由 key 和 pathname。
// 不去掉标识中的空白，空白字符串仍属于长度大于零的有效输入。
export function getNavigationVisitKey(location: NavigationVisitLocation) {
  if (isRecord(location.state)) {
    const visitId = location.state[NAVIGATION_VISIT_ID_KEY]
    if (typeof visitId === 'string' && visitId.length > 0) {
      return visitId
    }
  }

  return location.key || location.pathname
}
