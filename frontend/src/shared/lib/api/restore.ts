import { fetchWithAuth } from '@/shared/lib/api/client'
import { API_BASE_URL } from '@/shared/lib/config'

const RESTORE_URL = '/api/admin/restore'

export interface RestoreStatus {
  running: boolean
  phase: string | null
  done: boolean
  error?: string
  versionWarning?: string
  needsIntervention?: boolean
}

export interface RestoreReceipt {
  statusToken: string
}

export async function triggerRestore(id: string): Promise<RestoreReceipt> {
  return await fetchWithAuth<RestoreReceipt>(
    `${RESTORE_URL}/${encodeURIComponent(id)}`,
    {
      method: 'POST',
    },
  )
}

export async function triggerRestoreFromUpload(
  file: File,
): Promise<RestoreReceipt> {
  const formData = new FormData()
  formData.append('file', file)
  return await fetchWithAuth<RestoreReceipt>(`${RESTORE_URL}/upload`, {
    method: 'POST',
    body: formData,
    headers: {}, // Let browser set Content-Type for FormData
  })
}

// A job-scoped receipt survives account replacement without refreshing old sessions.
export async function getRestoreStatus(
  statusToken: string,
): Promise<RestoreStatus> {
  const response = await fetch(`${API_BASE_URL}/api/restore/progress`, {
    headers: { 'X-Restore-Token': statusToken },
    credentials: 'omit',
  })
  if (!response.ok) throw new Error(`Restore progress: HTTP ${response.status}`)
  return await response.json()
}

// triggerSelfRestart's request is expected to never receive a clean
// response: the server process replaces itself (Unix) or exits (Windows)
// mid-handler, dropping the connection. Callers should treat any rejection
// from this call as an unremarkable, expected outcome — not a real failure —
// and move on to waiting for the server to come back up.
export async function triggerSelfRestart(): Promise<void> {
  await fetchWithAuth(`${RESTORE_URL}/self-restart`, {
    method: 'POST',
  })
}
