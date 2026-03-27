<script lang="ts">
	import { browser } from '$app/environment';
	import { LineChart } from 'layerchart';
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

	// 圖表用：同一天只取最新一筆（metrics 已 DESC，取第一個 key 出現者）
	const chartData = $derived.by(() => {
		const seen = new Set<string>();
		return [...data.metrics]
			.filter(m => {
				const key = new Date(m.recorded_at).toLocaleDateString('en-CA');
				if (seen.has(key)) return false;
				seen.add(key);
				return true;
			})
			.reverse(); // 轉成 ASC 供圖表 x 軸使用
	});

	const abnormalDates = $derived(
		new Set(
			data.sleepLogs
				.filter(l => l.abnormal_wake)
				.map(l => new Date(l.wake_at).toLocaleDateString('en-CA'))
		)
	);

	const stepsMap = $derived(
		new Map(
			data.activities
				.filter(a => a.steps != null)
				.map(a => [a.activity_date, a.steps as number])
		)
	);

	const maxSteps = $derived(Math.max(1, ...[...stepsMap.values()]));

	const chartDataWithMeta = $derived(
		chartData.map(m => {
			const dateKey = new Date(m.recorded_at).toLocaleDateString('en-CA');
			return {
				...m,
				isAbnormal: abnormalDates.has(dateKey),
				steps: stepsMap.get(dateKey) ?? 0,
			};
		})
	);
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
{#if browser}
	<div class="rounded-2xl bg-white overflow-hidden shadow-md mb-8">
		<div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-gray-100">
			<div class="flex items-baseline gap-2">
				<span class="font-black tracking-widest text-sm text-gray-800">TRENDS</span>
				<span class="text-gray-400 text-xs">近 30 天</span>
			</div>
			<div class="flex items-center gap-4 text-xs text-gray-500">
				<span class="flex items-center gap-1.5">
					<span class="w-2.5 h-2.5 rounded-full bg-[#0EA5E9]"></span>體重
				</span>
				<span class="flex items-center gap-1.5">
					<span class="w-2.5 h-2.5 rounded-full bg-[#F59E0B]"></span>體脂率
				</span>
				<span class="flex items-center gap-1.5">
					<span class="w-2.5 h-2.5 rounded-full bg-[#10B981]"></span>肌肉率
				</span>
			</div>
		</div>

		{#if chartData.length >= 2}
			<div class="h-[240px]">
				<LineChart
					data={chartData}
					x={(d: BodyMetric) => new Date(d.recorded_at)}
					series={[
						{ key: 'weight_kg', label: '體重 (kg)', color: '#0EA5E9' },
						{ key: 'body_fat_pct', label: '體脂率 (%)', color: '#F59E0B' },
						{ key: 'muscle_pct', label: '肌肉率 (%)', color: '#10B981' },
					]}
					props={{
						tooltip: {
							root: {
								class: 'bg-white border border-gray-200 shadow-lg rounded-lg text-xs pointer-events-none',
							},
						},
					}}
				/>
			</div>

			<!-- 異常睡眠標記條 -->
			<div class="relative h-6 px-10 flex items-center">
				{#each chartDataWithMeta as point}
					{@const dates = chartDataWithMeta.map(d => new Date(d.recorded_at).getTime())}
					{@const minT = Math.min(...dates)}
					{@const maxT = Math.max(...dates)}
					{@const curT = new Date(point.recorded_at).getTime()}
					{@const pct = maxT === minT ? 50 : ((curT - minT) / (maxT - minT)) * 100}
					{#if point.isAbnormal}
						<span
							class="absolute text-orange-500 text-[10px] leading-none -translate-x-1/2"
							style="left: {pct}%"
							title="異常喚醒 {new Date(point.recorded_at).toLocaleDateString('zh-TW')}"
						>▲</span>
					{/if}
				{/each}
			</div>

			<!-- 步數熱度條 -->
			<div class="relative h-3 px-10 flex items-stretch mb-1 rounded-b overflow-hidden">
				{#each chartDataWithMeta as point}
					{@const dates = chartDataWithMeta.map(d => new Date(d.recorded_at).getTime())}
					{@const minT = Math.min(...dates)}
					{@const maxT = Math.max(...dates)}
					{@const curT = new Date(point.recorded_at).getTime()}
					{@const pct = maxT === minT ? 50 : ((curT - minT) / (maxT - minT)) * 100}
					{@const opacity = point.steps > 0 ? 0.15 + (point.steps / maxSteps) * 0.6 : 0}
					<div
						class="absolute inset-y-0 w-2 -translate-x-1/2 bg-emerald-500 rounded-sm"
						style="left: {pct}%; opacity: {opacity}"
						title="步數 {point.steps.toLocaleString()}"
					></div>
				{/each}
			</div>
		{:else}
			<div class="h-[200px] flex flex-col items-center justify-center gap-1 text-gray-400">
				<p class="text-sm font-medium">資料不足</p>
				<p class="text-xs">新增 2 筆以上體位數據即可查看趨勢</p>
			</div>
		{/if}
	</div>
{/if}
