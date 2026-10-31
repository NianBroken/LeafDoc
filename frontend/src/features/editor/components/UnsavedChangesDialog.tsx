// 未保存内容弹窗把继续离开与取消交还编辑器，组件自身不执行保存或页面导航。
import { DIALOG_UNSAVED_CHANGES } from '@/shared/lib/registries/instances'
import { useTranslation } from 'react-i18next'
import BaseDialog from '@/shared/components/BaseDialog'

type UnsavedChangesDialogProps = {
  onConfirm: () => void
  onCancel: () => void
}

// 关闭调用 onCancel，确认调用 onConfirm，回调完成后均返回 true 交给基础弹窗关闭。
// 取消按钮默认获取焦点，继续离开按钮使用破坏性样式提示将舍弃当前未保存内容。
export function UnsavedChangesDialog({
  onConfirm,
  onCancel,
}: UnsavedChangesDialogProps) {
  const { t } = useTranslation('common')
  return (
    <BaseDialog
      dialogTitle={t('unsavedChangesDialog.title')}
      dialogDescription={t('unsavedChangesDialog.description')}
      dialogType={DIALOG_UNSAVED_CHANGES}
      testidPrefix="unsaved-changes-dialog"
      onClose={() => {
        onCancel()
        return true
      }}
      onConfirm={async () => {
        onConfirm()
        return true
      }}
      cancelButton={{
        label: t('unsavedChangesDialog.cancelButton'),
        variant: 'secondary',
        autoFocus: true,
      }}
      buttons={[
        {
          label: t('unsavedChangesDialog.leaveAnywayButton'),
          variant: 'destructive',
          actionType: 'confirm',
          disabled: false,
          loading: false,
        },
      ]}
    ></BaseDialog>
  )
}
