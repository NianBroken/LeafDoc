// 此按钮只打开全局密钥表单，对话框负责收集输入和提交创建请求。
import { Button } from '@/shared/components/ui/button'
import { DIALOG_API_KEY_FORM } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTranslation } from 'react-i18next'

// 不传入对话框参数，表单从自己的初始状态与用户存储读取所需资料。
export function CreateApiKeyButton() {
  const { t } = useTranslation('apikeys')
  const openDialog = useDialogsStore((s) => s.openDialog)
  return (
    <Button variant="default" onClick={() => openDialog(DIALOG_API_KEY_FORM)}>
      {t('create.button')}
    </Button>
  )
}
