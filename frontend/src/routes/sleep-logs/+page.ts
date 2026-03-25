import { listSleepLogs } from '$lib/api/sleep-logs'
import type { PageLoad } from './$types'

export const load: PageLoad = async () => {
  try {
    const res = await listSleepLogs()
    return { logs: res.data, meta: res.meta }
  } catch {
    return { logs: [], meta: { total: 0 } }
  }
}
