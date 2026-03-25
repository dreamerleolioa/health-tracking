import { listBodyMetrics } from '$lib/api/body-metrics';
import { listSleepLogs } from '$lib/api/sleep-logs';
import { listDailyActivities } from '$lib/api/daily-activities';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const [metricsRes, sleepRes, activityRes] = await Promise.allSettled([
		listBodyMetrics({ limit: 90 }, fetch),
		listSleepLogs(),
		listDailyActivities(),
	]);

	return {
		metrics: metricsRes.status === 'fulfilled' ? metricsRes.value.data : [],
		meta: metricsRes.status === 'fulfilled' ? metricsRes.value.meta : { total: 0 },
		sleepLogs: sleepRes.status === 'fulfilled' ? sleepRes.value.data : [],
		activities: activityRes.status === 'fulfilled' ? activityRes.value.data : [],
	};
};
