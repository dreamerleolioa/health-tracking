import { listDailyActivities } from '$lib/api/daily-activities'
import type { PageLoad } from './$types'

export const load: PageLoad = async () => {
  try {
    const res = await listDailyActivities()
    return { activities: res.data, meta: res.meta }
  } catch {
    return { activities: [], meta: { total: 0 } }
  }
}
