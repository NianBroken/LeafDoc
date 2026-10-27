import { Button } from '@/shared/components/ui/button'
import { mapApiError } from '@/shared/lib/api/errors'
import { User } from '@/shared/lib/api/users'
import { useUserStore } from '@/features/users/state/users'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

type ResendInviteButtonProps = {
  user: User
}

export function ResendInviteButton({ user }: ResendInviteButtonProps) {
  const { t } = useTranslation('users')
  const resendInvite = useUserStore((s) => s.resendInvite)
  const [loading, setLoading] = useState(false)

  const handleClick = async () => {
    setLoading(true)
    try {
      await resendInvite(user.id)
      toast.success(t('invite.resendSuccessToast'))
    } catch (err) {
      console.warn(err)
      const mapped = mapApiError(err, t('invite.resendErrorFallback'))
      toast.error(mapped.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Button
      size="sm"
      variant="secondary"
      disabled={loading}
      onClick={handleClick}
    >
      {loading ? t('invite.resending') : t('invite.resendAction')}
    </Button>
  )
}
