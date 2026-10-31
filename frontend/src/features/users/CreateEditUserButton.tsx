// 用户表单入口根据是否传入用户选择编辑或创建，按钮本身不发送用户保存请求。
import { DropdownMenuItem } from '@/shared/components/ui/dropdown-menu'
import { Button } from '@/shared/components/ui/button'
import { User } from '@/shared/lib/api/users'
import { DIALOG_USER_FORM } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTranslation } from 'react-i18next'
type CreateEditUserButtonProps = {
  user?: User
  menuItem?: boolean
}

// 已有用户且指定 menuItem 时使用菜单项，其余编辑入口为小号描边按钮。
// 没有用户时始终显示创建按钮，即使 menuItem 为真也不生成菜单项。
export function CreateEditUserButton({
  user,
  menuItem,
}: CreateEditUserButtonProps) {
  const { t } = useTranslation('users')
  const openDialog = useDialogsStore((s) => s.openDialog)
  if (user && menuItem) {
    return (
      <DropdownMenuItem onSelect={() => openDialog(DIALOG_USER_FORM, { user })}>
        {t('actions.edit')}
      </DropdownMenuItem>
    )
  }
  return user ? (
    <Button
      size="sm"
      variant="outline"
      onClick={() => openDialog(DIALOG_USER_FORM, { user })}
    >
      {t('actions.edit')}
    </Button>
  ) : (
    <Button variant="default" onClick={() => openDialog(DIALOG_USER_FORM)}>
      {t('actions.new')}
    </Button>
  )
}
