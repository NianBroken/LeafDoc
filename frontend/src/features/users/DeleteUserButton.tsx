// 用户删除入口只打开确认弹窗，传入账号标识和显示名，由弹窗提交删除请求。
import { Button } from '@/shared/components/ui/button'
import { User } from '@/shared/lib/api/users'
import { DIALOG_DELETE_USER_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useSessionStore } from '@/shared/state/session'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem } from '@/shared/components/ui/dropdown-menu'

type DeleteUserButtonProps = {
  user: User
  menuItem?: boolean
}

// 当前会话 ID 与目标相同时不展示入口，其他目标按 menuItem 选择菜单项或按钮。
// 此显示限制不是服务端权限校验，未取得当前会话用户时也不会在此分支隐藏按钮。
export function DeleteUserButton({ user, menuItem }: DeleteUserButtonProps) {
  const { t } = useTranslation('users')
  const openDialog = useDialogsStore((s) => s.openDialog)

  const { user: currentUser } = useSessionStore()
  if (currentUser?.id === user.id) return null

  const handleClick = () =>
    openDialog(DIALOG_DELETE_USER_CONFIRMATION, {
      userId: user.id,
      username: user.username,
    })

  if (menuItem) {
    return (
      <DropdownMenuItem className="text-error" onSelect={handleClick}>
        {t('actions.delete')}
      </DropdownMenuItem>
    )
  }

  return (
    <Button size="sm" variant="destructive" onClick={handleClick}>
      {t('actions.delete')}
    </Button>
  )
}
