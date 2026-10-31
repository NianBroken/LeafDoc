// 下拉菜单使用的预留高度，仅用来判断是否滚动触发器，不设置菜单实际高度。
const menuSpace = 320

// 菜单固定向下展开并关闭碰撞改向，空间不足由下方滚动辅助处理。
export const downwardSelectContentProps = {
  side: 'bottom',
  avoidCollisions: false,
} as const

// 仅在打开时检查触发器下方空间，不足预留值则将触发器立即滚到视口中部。
// 找不到元素或空间足够时不动作，不监听后续窗口变化。
export function revealDownwardSelect(triggerId: string, open: boolean): void {
  if (!open) return
  const trigger = document.getElementById(triggerId)
  if (
    trigger &&
    window.innerHeight - trigger.getBoundingClientRect().bottom < menuSpace
  ) {
    trigger.scrollIntoView({
      block: 'center',
      inline: 'nearest',
      behavior: 'instant',
    })
  }
}
