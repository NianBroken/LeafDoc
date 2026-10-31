// 此表单创建只读权限密钥，并在当前对话框内展示创建响应中的明文令牌。
// 用户列表来自外部已加载的用户存储，组件不在内部请求用户列表。
import BaseDialog from '@/shared/components/BaseDialog'
import { Button } from '@/shared/components/ui/button'
import { FormInput } from '@/shared/components/FormInput'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/components/ui/select'
import type { CreateApiKeyResult } from '@/shared/lib/api/apikeys'
import { handleFieldErrors } from '@/shared/lib/handleFieldErrors'
import { DIALOG_API_KEY_FORM } from '@/shared/lib/registries/instances'
import { useApiKeyStore } from '@/features/apikeys/state/apikeys'
import { useUserStore } from '@/features/users/state/users'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

// 名称输入框保留回车交给对话框快捷键处理，不放行其他页面快捷键。
const DIALOG_INPUT_ALLOWED_HOTKEYS = 'Enter'

// toExpiresAtRFC3339 将日期输入的本地日历日期转换为该日 23:59:59 的 UTC 时刻。
// 拼接的时间没有时区后缀，Date 按浏览器本地时区解释，再由 toISOString 输出 Z 格式。
// 空日期表示不设置到期时间；非空值不在这里验证日期或未来性，异常交给提交分支处理。
function toExpiresAtRFC3339(dateOnly: string): string | undefined {
  if (!dateOnly) return undefined
  return new Date(`${dateOnly}T23:59:59`).toISOString()
}

// ApiKeyFormDialog 在创建成功后切换为令牌展示状态，不自动关闭或再次提交。
// 名称、所属用户和到期日期各自保存，字段错误由公共映射器填入。
export function ApiKeyFormDialog() {
  const { t } = useTranslation('apikeys')
  const [name, setName] = useState('')
  const [userId, setUserId] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<CreateApiKeyResult | null>(null)

  const { createApiKey } = useApiKeyStore()
  const users = useUserStore((s) => s.users)

  const handleCreate = async (): Promise<boolean> => {
    if (!name || !userId) return false // 与按钮禁用条件一致，空输入不提交请求。

    setLoading(true)
    try {
      const created = await createApiKey({
        name,
        userId,
        // 请求不携带 role，核心服务为空角色采用 viewer，不从所选用户推导更高权限。
        expiresAt: toExpiresAtRFC3339(expiresAt),
      })
      toast.success(t('create.successToast'))
      setResult(created)
      return false // 保持打开，让用户查看并复制只在创建响应中返回的令牌。
    } catch (err) {
      console.warn(err)
      handleFieldErrors(err, setFieldErrors, t('create.errorFallback'))
      return false
    } finally {
      setLoading(false)
    }
  }

  const handleCopy = async () => {
    // 只复制已创建结果中的令牌，剪贴板权限或接口失败会展示独立错误提示。
    if (!result) return
    try {
      await navigator.clipboard.writeText(result.secret)
      toast.success(t('secret.copiedToast'))
    } catch (err) {
      console.warn(err)
      toast.error(t('secret.copyErrorToast'))
    }
  }

  return (
    <BaseDialog
      dialogType={DIALOG_API_KEY_FORM}
      dialogTitle={result ? t('secret.title') : t('create.title')}
      dialogDescription={
        result ? t('secret.description') : t('create.description')
      }
      onClose={() => true}
      onConfirm={handleCreate}
      testidPrefix="api-key-form-dialog"
      // 创建后将取消位置用作完成按钮，并隐藏提交按钮，避免重复创建。
      cancelButton={
        result
          ? { label: t('secret.done'), variant: 'default', autoFocus: true }
          : {
              label: t('create.cancel'),
              variant: 'outline',
              disabled: loading,
            }
      }
      buttons={
        result
          ? []
          : [
              {
                label: t('create.submit'),
                actionType: 'confirm',
                loading,
                disabled: loading || !name || !userId,
              },
            ]
      }
    >
      {/* 结果存在时仅展示令牌，未创建时显示输入控件，状态切换不依赖另一次列表查询。 */}
      {result ? (
        <div className="space-y-4 pt-2">
          <FormInput
            label={t('secret.label')}
            name="secret"
            value={result.secret}
            onChange={() => {}}
            readOnly
            testid="api-key-secret"
          />
          <Button
            type="button"
            variant="outline"
            onClick={handleCopy}
            data-testid="api-key-secret-copy"
          >
            {t('secret.copy')}
          </Button>
          <p className="text-muted text-sm">{t('secret.helper')}</p>
        </div>
      ) : (
        <div className="space-y-4 pt-2">
          <FormInput
            autoFocus={true}
            label={t('create.nameLabel')}
            name="name"
            value={name}
            onChange={(val) => {
              // 编辑名称时只清除名称错误，不清除其他字段的服务器校验结果。
              setName(val)
              setFieldErrors((prev) => ({ ...prev, name: '' }))
            }}
            placeholder={t('create.namePlaceholder')}
            error={fieldErrors.name}
            allowedHotkeys={DIALOG_INPUT_ALLOWED_HOTKEYS}
          />
          <Select
            value={userId}
            onValueChange={(val) => {
              setUserId(val)
              setFieldErrors((prev) => ({ ...prev, userId: '' }))
            }}
          >
            <SelectTrigger data-testid="api-key-owner-select">
              <SelectValue placeholder={t('create.userPlaceholder')} />
            </SelectTrigger>
            <SelectContent>
              {users.map((user) => (
                <SelectItem key={user.id} value={user.id}>
                  {user.username}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FormInput
            label={t('create.expiresLabel')}
            name="expiresAt"
            type="date"
            value={expiresAt}
            onChange={(val) => {
              setExpiresAt(val)
              setFieldErrors((prev) => ({ ...prev, expiresAt: '' }))
            }}
            error={fieldErrors.expiresAt}
            testid="api-key-expires-at"
          />
          <p className="text-muted text-sm">{t('create.roleHelper')}</p>
        </div>
      )}
    </BaseDialog>
  )
}
