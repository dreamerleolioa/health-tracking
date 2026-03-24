import { listBodyMetrics } from '$lib/api/body-metrics';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const res = await listBodyMetrics({ limit: 90 }, fetch);
		return { metrics: res.data, meta: res.meta };
	} catch {
		return { metrics: [], meta: { total: 0 } };
	}
};
