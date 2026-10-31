// 此按钮将选中密钥的标识和名称交给撤销确认框，不直接调用删除接口。
import { TrashIcon } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import type { ApiKey } from '@/shared/lib/api/apikeys'
import { DIALOG_DELETE_API_KEY_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTranslation } from 'react-i18next'

// 接收完整公开记录，标识用于提交，名称用于确认文字。
type DeleteApiKeyButtonProps = {
  apiKey: ApiKey
}

// 图标按钮同时提供无障碍名称与悬停标题，两者使用相同翻译。
export function DeleteApiKeyButton({ apiKey }: DeleteApiKeyButtonProps) {
  const { t } = useTranslation('apikeys')
  const openDialog = useDialogsStore((s) => s.openDialog)

  return (
    <Button
      size="icon"
      className="h-7 w-7"
      aria-label={t('delete.button')}
      title={t('delete.button')}
      variant="destructive"
      onClick={() =>
        openDialog(DIALOG_DELETE_API_KEY_CONFIRMATION, {
          apiKeyId: apiKey.id,
          apiKeyName: apiKey.name,
        })
      }
    >
      <TrashIcon className="h-4 w-4" />
    </Button>
  )
}
