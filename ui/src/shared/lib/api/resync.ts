import { fetchWithAuth } from '@/shared/lib/api/client'

const RESYNC_URL = '/api/admin/resync'

export interface ResyncStatus {
  running: boolean
  phase: string | null
  done: boolean
  error?: string
}

export async function triggerResync(): Promise<void> {
  await fetchWithAuth(RESYNC_URL, {
    method: 'POST',
  })
}

export async function getResyncStatus(): Promise<ResyncStatus> {
  return await fetchWithAuth<ResyncStatus>(`${RESYNC_URL}/status`, {})
}
