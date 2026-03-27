<script lang="ts">
	import type { PageData } from './$types';
	import type { BodyMetric } from '$lib/types';

	let { data }: { data: PageData } = $props();

	const today = new Date().toLocaleDateString('zh-TW', {
		year: 'numeric',
		month: 'long',
		day: 'numeric',
		weekday: 'long',
	});

	const todayStr = new Date().toISOString().slice(0, 10);

	// 最新一筆 body metric（API 回傳 DESC，第一筆最新）
	const latest = $derived(data.metrics[0] as BodyMetric | undefined);

	const metricCards = $derived([
		{
			label: '體重',
			emoji: '⚖️',
			value: latest?.weight_kg != null ? String(latest.weight_kg) : '—',
			unit: latest?.weight_kg != null ? 'kg' : '',
			prefix: '',
			color: '#0EA5E9',
		},
		{
			label: '體脂率',
			emoji: '🔥',
			value: latest?.body_fat_pct != null ? String(latest.body_fat_pct) : '—',
			unit: latest?.body_fat_pct != null ? '%' : '',
			prefix: '',
			color: '#F59E0B',
		},
		{
			label: '肌肉率',
			emoji: '💪',
			value: latest?.muscle_pct != null ? String(latest.muscle_pct) : '—',
			unit: latest?.muscle_pct != null ? '%' : '',
			prefix: '',
			color: '#10B981',
		},
		{
			label: '內臟脂肪',
			emoji: '📊',
			value: latest?.visceral_fat != null ? String(latest.visceral_fat) : '—',
			unit: '',
			prefix: latest?.visceral_fat != null ? 'Lv.' : '',
			color: '#8B5CF6',
		},
	]);

	// 今日活動步數
	const todayActivity = $derived(
		data.activities.find(a => a.activity_date === todayStr)
	);

	// 最近睡眠
	const latestSleep = $derived(data.sleepLogs[0]);
</script>

<!-- 頁面標題 -->
<div class="mb-8">
	<h1 class="text-white font-black tracking-widest text-2xl">TODAY</h1>
	<p class="text-gray-400 text-sm mt-1">{today}</p>
</div>

<!-- 數據卡片 -->
<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
	{#each metricCards as metric}
		<div
			class="bg-white rounded-2xl overflow-hidden shadow-md hover:-translate-y-1 hover:shadow-xl transition-all duration-150 cursor-pointer"
		>
			<!-- 頂部色條 -->
			<div class="h-1" style="background-color: {metric.color};"></div>

			<div class="p-5">
				<!-- Emoji -->
				<div class="text-3xl mb-3">{metric.emoji}</div>

				<!-- 數值 + 單位 -->
				<div class="flex items-end gap-1 mb-2">
					{#if metric.prefix}
						<span class="text-base text-gray-400 mb-1">{metric.prefix}</span>
					{/if}
					<span class="text-4xl font-black text-gray-900 leading-none">{metric.value}</span>
					{#if metric.unit}
						<span class="text-base text-gray-400 mb-1">{metric.unit}</span>
					{/if}
				</div>

				<!-- 標籤 -->
				<p class="text-xs text-gray-500 font-medium tracking-wide">{metric.label}</p>
			</div>
		</div>
	{/each}
</div>

<!-- 今日摘要 -->
{#if todayActivity || latestSleep}
	<div class="grid grid-cols-2 gap-4 mb-8">
		{#if latestSleep}
			<div class="bg-white/5 rounded-2xl p-4">
				<p class="text-gray-400 text-xs tracking-wide mb-2">上次睡眠</p>
				<p class="text-white font-bold text-lg">
					{latestSleep.duration_min != null
						? `${Math.floor(latestSleep.duration_min / 60)}h ${latestSleep.duration_min % 60}m`
						: '—'}
				</p>
				{#if latestSleep.abnormal_wake}
					<p class="text-orange-400 text-xs mt-1">▲ 異常喚醒</p>
				{/if}
			</div>
		{/if}
		{#if todayActivity?.steps != null}
			<div class="bg-white/5 rounded-2xl p-4">
				<p class="text-gray-400 text-xs tracking-wide mb-2">今日步數</p>
				<p class="text-white font-bold text-lg">{todayActivity.steps.toLocaleString()}</p>
				<p class="text-gray-500 text-xs mt-1">步</p>
			</div>
		{/if}
	</div>
{/if}

<!-- 趨勢圖區 -->
<div class="rounded-2xl p-6 bg-white/5">
	<div class="flex items-baseline gap-3 mb-4">
		<h2 class="text-white font-black tracking-widest text-lg">TRENDS</h2>
		<span class="text-gray-400 text-sm">近 30 天</span>
	</div>

	<!-- 圖表佔位 -->
	<div
		class="h-48 rounded-xl flex items-center justify-center bg-white/5"
	>
		<p class="text-gray-500 text-sm">圖表載入中...</p>
	</div>
</div>
