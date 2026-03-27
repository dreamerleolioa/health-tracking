import { listBodyMetrics } from '$lib/api/body-metrics';
import { listSleepLogs } from '$lib/api/sleep-logs';
import { listDailyActivities } from '$lib/api/daily-activities';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  const today = new Date();
  const from = new Date(today);
  from.setDate(today.getDate() - 30);
  const fromStr = from.toISOString().slice(0, 10);
  const toStr = today.toISOString().slice(0, 10);

  const [metricsRes, sleepRes, activityRes] = await Promise.allSettled([
    listBodyMetrics({ from: fromStr, to: toStr, limit: 90 }, fetch),
    listSleepLogs({ from: fromStr, to: toStr }, fetch),
    listDailyActivities({ from: fromStr, to: toStr }, fetch),
  ]);

  return {
    metrics: metricsRes.status === 'fulfilled' ? metricsRes.value.data : [],
    sleepLogs: sleepRes.status === 'fulfilled' ? sleepRes.value.data : [],
    activities: activityRes.status === 'fulfilled' ? activityRes.value.data : [],
  };
};
