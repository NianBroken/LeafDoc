// 此对话框提交密钥撤销并等待列表刷新，成功后关闭，失败时保留确认内容。
import BaseDialog from '@/shared/components/BaseDialog'
import { mapApiError } from '@/shared/lib/api/errors'
import { DIALOG_DELETE_API_KEY_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useApiKeyStore } from '@/features/apikeys/state/apikeys'
import { useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { toast } from 'sonner'

// 标识是写请求目标，名称只参与翻译插值，不作为后端查找条件。
type DeleteApiKeyDialogProps = {
  apiKeyId: string
  apiKeyName: string
}

// 默认键盘动作设为取消，降低误触确认的机会；loading 控制按钮可用性。
export function DeleteApiKeyDialog({
  apiKeyId,
  apiKeyName,
}: DeleteApiKeyDialogProps) {
  const { t } = useTranslation('apikeys')
  const { deleteApiKey } = useApiKeyStore()

  const [loading, setLoading] = useState(false)

  const handleRevoke = async (): Promise<boolean> => {
    // 存储动作包含列表重载，任何一步失败都映射为提示并保持对话框打开。
    setLoading(true)
    try {
      await deleteApiKey(apiKeyId)
      toast.success(t('delete.successToast'))
      return true // BaseDialog 根据 true 关闭当前确认框。
    } catch (err) {
      console.error('Error revoking API key:', err)
      const mapped = mapApiError(err, t('delete.errorFallback'))
      toast.error(mapped.message)
      return false // 错误已经展示，继续保留确认框供用户处理。
    } finally {
      // 无论成功或失败都结束按钮加载状态，不改变已返回的关闭决定。
      setLoading(false)
    }
  }

  return (
    <BaseDialog
      dialogType={DIALOG_DELETE_API_KEY_CONFIRMATION}
      dialogTitle={t('delete.title')}
      dialogDescription={t('delete.description')}
      onClose={() => true}
      onConfirm={async (): Promise<boolean> => {
        return await handleRevoke()
      }}
      defaultAction="cancel"
      testidPrefix="delete-api-key-dialog"
      cancelButton={{
        label: t('delete.cancel'),
        variant: 'outline',
        disabled: loading,
        autoFocus: true,
      }}
      buttons={[
        {
          label: loading ? t('delete.confirming') : t('delete.confirm'),
          actionType: 'confirm',
          variant: 'destructive',
          autoFocus: false,
          loading,
          disabled: loading,
        },
      ]}
    >
      <p className="text-muted text-sm">
        <Trans
          i18nKey="delete.body"
          ns="apikeys"
          values={{ name: apiKeyName }}
          components={{ strong: <strong /> }}
        />
      </p>
    </BaseDialog>
  )
}
