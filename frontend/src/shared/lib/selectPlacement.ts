// Downward menus keep the trigger visible and reserve space for a scrollable list.
const menuSpace = 320

export const downwardSelectContentProps = {
  side: 'bottom',
  avoidCollisions: false,
} as const

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
