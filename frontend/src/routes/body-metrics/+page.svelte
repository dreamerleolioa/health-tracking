<script lang="ts">
	import { browser } from '$app/environment';
	import { invalidateAll } from '$app/navigation';
	import { slide, fade, fly } from 'svelte/transition';
	import { LineChart } from 'layerchart';
	import { createBodyMetric, updateBodyMetric, deleteBodyMetric } from '$lib/api/body-metrics';
	import type { BodyMetric, SleepLog, DailyActivity } from '$lib/types';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	// abnormalDates: Set of YYYY-MM-DD strings where abnormal_wake = true
	const abnormalDates = $derived(
		new Set(
			(data.sleepLogs as SleepLog[])
				.filter((l) => l.abnormal_wake)
				.map((l) => new Date(l.wake_at).toLocaleDateString('en-CA'))
		)
	);

	// stepsMap: activity_date → steps count
	const stepsMap = $derived(
		new Map(
			(data.activities as DailyActivity[])
				.filter((a) => a.steps != null)
				.map((a) => [a.activity_date, a.steps as number])
		)
	);

	const maxSteps = $derived(Math.max(1, ...[...stepsMap.values()]));

	// Dedup: data is ORDER BY recorded_at DESC → first per day = latest
	const chartData = $derived.by(() => {
		const seen = new Set<string>();
		return (data.metrics as BodyMetric[])
			.filter((m) => {
				const key = new Date(m.recorded_at).toLocaleDateString('en-CA');
				if (seen.has(key)) return false;
				seen.add(key);
				return true;
			})
			.reverse();
	});

	let showForm = $state(false);
	let submitting = $state(false);
	let deleting = $state(new Set<string>());

	let recorded_at = $state('');
	let weight_kg = $state('');
	let body_fat_pct = $state('');
	let muscle_pct = $state('');
	let visceral_fat = $state('');
	let note = $state('');

	function defaultRecordedAt() {
		const now = new Date();
		const offset = now.getTimezoneOffset() * 60000;
		return new Date(now.getTime() - offset).toISOString().slice(0, 16);
	}

	function openForm() {
		recorded_at = defaultRecordedAt();
		weight_kg = '';
		body_fat_pct = '';
		muscle_pct = '';
		visceral_fat = '';
		note = '';
		showForm = true;
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		submitting = true;
		try {
			await createBodyMetric({
				recorded_at: new Date(recorded_at).toISOString(),
				...(weight_kg && { weight_kg: parseFloat(weight_kg) }),
				...(body_fat_pct && { body_fat_pct: parseFloat(body_fat_pct) }),
				...(muscle_pct && { muscle_pct: parseFloat(muscle_pct) }),
				...(visceral_fat && { visceral_fat: parseInt(visceral_fat) }),
				...(note && { note })
			});
			showForm = false;
			await invalidateAll();
		} catch (err) {
			alert('新增失敗：' + (err instanceof Error ? err.message : '未知錯誤'));
		} finally {
			submitting = false;
		}
	}

	// ── Delete with confirmation ──────────────────────
	let confirmDeleteId = $state<string | null>(null);

	async function executeDelete() {
		if (!confirmDeleteId) return;
		const id = confirmDeleteId;
		confirmDeleteId = null;
		deleting = new Set([...deleting, id]);
		try {
			await deleteBodyMetric(id);
			await invalidateAll();
		} catch (err) {
			alert('刪除失敗：' + (err instanceof Error ? err.message : '未知錯誤'));
		} finally {
			deleting = new Set([...deleting].filter((d) => d !== id));
		}
	}

	// ── Edit ──────────────────────────────────────────
	let editingMetric = $state<BodyMetric | null>(null);
	let editSubmitting = $state(false);
	let e_recorded_at = $state('');
	let e_weight_kg = $state('');
	let e_body_fat_pct = $state('');
	let e_muscle_pct = $state('');
	let e_visceral_fat = $state('');
	let e_note = $state('');

	function toLocalInput(isoStr: string) {
		const d = new Date(isoStr);
		return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
	}

	function openEdit(m: BodyMetric) {
		editingMetric = m;
		e_recorded_at = toLocalInput(m.recorded_at);
		e_weight_kg = m.weight_kg != null ? String(m.weight_kg) : '';
		e_body_fat_pct = m.body_fat_pct != null ? String(m.body_fat_pct) : '';
		e_muscle_pct = m.muscle_pct != null ? String(m.muscle_pct) : '';
		e_visceral_fat = m.visceral_fat != null ? String(m.visceral_fat) : '';
		e_note = m.note ?? '';
	}

	async function handleEditSubmit(ev: SubmitEvent) {
		ev.preventDefault();
		if (!editingMetric) return;
		editSubmitting = true;
		try {
			await updateBodyMetric(editingMetric.id, {
				recorded_at: new Date(e_recorded_at).toISOString(),
				...(e_weight_kg && { weight_kg: parseFloat(e_weight_kg) }),
				...(e_body_fat_pct && { body_fat_pct: parseFloat(e_body_fat_pct) }),
				...(e_muscle_pct && { muscle_pct: parseFloat(e_muscle_pct) }),
				...(e_visceral_fat && { visceral_fat: parseInt(e_visceral_fat) }),
				...(e_note && { note: e_note })
			});
			editingMetric = null;
			await invalidateAll();
		} catch (err) {
			alert('編輯失敗：' + (err instanceof Error ? err.message : '未知錯誤'));
		} finally {
			editSubmitting = false;
		}
	}

	function formatDate(isoStr: string) {
		const d = new Date(isoStr);
		const mm = String(d.getMonth() + 1).padStart(2, '0');
		const dd = String(d.getDate()).padStart(2, '0');
		const hh = String(d.getHours()).padStart(2, '0');
		const min = String(d.getMinutes()).padStart(2, '0');
		return `${mm}/${dd} ${hh}:${min}`;
	}

	type Delta = 'up' | 'down' | 'same' | 'none';

	function delta(current: number | null | undefined, prev: number | null | undefined): Delta {
		if (current == null || prev == null) return 'none';
		if (current > prev) return 'up';
		if (current < prev) return 'down';
		return 'same';
	}

	// chartDataWithMeta: extend chartData with abnormal + steps info
	const chartDataWithMeta = $derived(
		chartData.map((m) => {
			const dateKey = new Date(m.recorded_at).toLocaleDateString('en-CA');
			return {
				...m,
				isAbnormal: abnormalDates.has(dateKey),
				steps: stepsMap.get(dateKey) ?? 0
			};
		})
	);

	// higherIsBetter = true for muscle_pct; false for weight/body_fat/visceral_fat
	function deltaClass(d: Delta, higherIsBetter: boolean): string {
		if (d === 'none' || d === 'same') return 'text-gray-800';
		const good = higherIsBetter ? d === 'up' : d === 'down';
		return good ? 'text-emerald-500' : 'text-red-500';
	}
</script>

<!-- ── Header ───────────────────────────────────────── -->
<div class="flex items-center justify-between mb-5">
	<div>
		<h1 class="text-white font-black tracking-widest text-xl">體位數據</h1>
		<p class="text-white/40 text-xs mt-0.5">共 {data.meta.total} 筆記錄</p>
	</div>
	<button
		onclick={openForm}
		class="bg-white text-gray-900 text-sm font-bold px-4 py-2 rounded-lg hover:bg-gray-100 transition-colors"
	>
		+ 新增記錄
	</button>
</div>

<!-- ── Trends Chart ──────────────────────────────────── -->
{#if browser}
	<div class="mb-5 bg-white rounded-xl shadow-sm overflow-hidden">
		<!-- card header -->
		<div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-gray-100">
			<div class="flex items-baseline gap-2">
				<span class="font-black tracking-widest text-sm text-gray-800">TRENDS</span>
				<span class="text-gray-400 text-xs">近 90 天</span>
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

		<!-- chart or placeholder -->
		{#if chartData.length >= 2}
			<div class="h-[240px]">
				<LineChart
					data={chartData}
					x={(d: BodyMetric) => new Date(d.recorded_at)}
					series={[
						{ key: 'weight_kg', label: '體重 (kg)', color: '#0EA5E9' },
						{ key: 'body_fat_pct', label: '體脂率 (%)', color: '#F59E0B' },
						{ key: 'muscle_pct', label: '肌肉率 (%)', color: '#10B981' }
					]}
					props={{
						tooltip: {
							root: {
								class:
									'bg-white border border-gray-200 shadow-lg rounded-lg text-xs pointer-events-none'
							}
						}
					}}
				/>
			</div>

			<!-- Abnormal sleep marker strip -->
			<div class="relative h-6 px-10 flex items-center">
				{#each chartDataWithMeta as point}
					{@const dates = chartDataWithMeta.map((d) => new Date(d.recorded_at).getTime())}
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

			<!-- Steps heatmap strip -->
			<div class="relative h-3 px-10 flex items-stretch mb-1 rounded-b overflow-hidden">
				{#each chartDataWithMeta as point}
					{@const dates = chartDataWithMeta.map((d) => new Date(d.recorded_at).getTime())}
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
			<div class="h-[160px] flex flex-col items-center justify-center gap-1 text-gray-400">
				<p class="text-sm font-medium">資料不足</p>
				<p class="text-xs">新增 2 筆以上記錄即可查看趨勢</p>
			</div>
		{/if}
	</div>
{/if}

<!-- ── Inline Form ───────────────────────────────────── -->
{#if showForm}
	<div transition:slide={{ duration: 180 }} class="mb-5 bg-white rounded-xl shadow-sm">
		<div class="px-5 pt-4 pb-3 border-b border-gray-100">
			<span class="font-black tracking-widest text-sm text-gray-800">新增記錄</span>
		</div>
		<form onsubmit={handleSubmit} class="p-5 grid grid-cols-2 md:grid-cols-4 gap-4">
			<!-- recorded_at spans full width -->
			<div class="col-span-2 md:col-span-4">
				<label for="recorded_at" class="block text-xs text-gray-400 mb-1">記錄時間 *</label>
				<input
					id="recorded_at"
					type="datetime-local"
					bind:value={recorded_at}
					required
					class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
				/>
			</div>

			<!-- numeric fields 2×2 -->
			<div>
				<label for="weight_kg" class="block text-xs text-gray-400 mb-1">體重 (kg)</label>
				<input
					id="weight_kg"
					type="number"
					step="0.1"
					min="30"
					max="300"
					bind:value={weight_kg}
					placeholder="72.5"
					class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
				/>
			</div>
			<div>
				<label for="body_fat_pct" class="block text-xs text-gray-400 mb-1">體脂率 (%)</label>
				<input
					id="body_fat_pct"
					type="number"
					step="0.1"
					min="1"
					max="70"
					bind:value={body_fat_pct}
					placeholder="18.2"
					class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
				/>
			</div>
			<div>
				<label for="muscle_pct" class="block text-xs text-gray-400 mb-1">肌肉率 (%)</label>
				<input
					id="muscle_pct"
					type="number"
					step="0.1"
					min="10"
					max="80"
					bind:value={muscle_pct}
					placeholder="35.2"
					class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
				/>
			</div>
			<div>
				<label for="visceral_fat" class="block text-xs text-gray-400 mb-1">內臟脂肪</label>
				<input
					id="visceral_fat"
					type="number"
					min="1"
					max="30"
					bind:value={visceral_fat}
					placeholder="8"
					class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
				/>
			</div>

			<!-- note + actions -->
			<div class="col-span-2 md:col-span-4">
				<label for="note" class="block text-xs text-gray-400 mb-1">備註（選填）</label>
				<textarea
					id="note"
					bind:value={note}
					rows="2"
					placeholder="今天狀況…"
					class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 resize-none focus:outline-none focus:border-[#E4000F]"
				></textarea>
			</div>
			<div class="col-span-2 md:col-span-4 flex justify-end gap-2 pt-1">
				<button
					type="button"
					onclick={() => (showForm = false)}
					class="px-4 py-2 text-xs text-gray-400 hover:text-gray-600 transition-colors"
				>
					取消
				</button>
				<button
					type="submit"
					disabled={submitting}
					class="bg-gray-900 text-white text-xs font-bold px-5 py-2 rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-40"
				>
					{submitting ? '儲存中…' : '儲存'}
				</button>
			</div>
		</form>
	</div>
{/if}

<!-- ── Data Table ────────────────────────────────────── -->
{#if data.metrics.length === 0}
	<div class="flex flex-col items-center justify-center py-20 text-white/30 gap-1">
		<p class="text-sm">尚無紀錄</p>
		<p class="text-xs">點擊「+ 新增記錄」開始追蹤</p>
	</div>
{:else}
	<!-- ── Desktop Table (md+) ── -->
	<div class="hidden md:block bg-white rounded-xl shadow-sm overflow-hidden">
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wider">
					<th class="text-left px-5 py-3 font-medium">日期</th>
					<th class="text-right px-4 py-3 font-medium">體重</th>
					<th class="text-right px-4 py-3 font-medium">體脂率</th>
					<th class="text-right px-4 py-3 font-medium">肌肉率</th>
					<th class="text-right px-4 py-3 font-medium">內臟脂肪</th>
					<th class="w-20 px-4 py-3"></th>
				</tr>
			</thead>
			<tbody>
				{#each data.metrics as m, i (m.id)}
					{@const prev = data.metrics[i + 1]}
					{@const wClass = deltaClass(delta(m.weight_kg, prev?.weight_kg), false)}
					{@const fClass = deltaClass(delta(m.body_fat_pct, prev?.body_fat_pct), false)}
					{@const mClass = deltaClass(delta(m.muscle_pct, prev?.muscle_pct), true)}
					{@const vClass = deltaClass(delta(m.visceral_fat, prev?.visceral_fat), false)}
					<tr class="border-b border-gray-50 last:border-0 transition-colors group hover:bg-gray-100 relative [&>td:first-child]:relative [&>td:first-child]:before:absolute [&>td:first-child]:before:left-0 [&>td:first-child]:before:top-0 [&>td:first-child]:before:bottom-0 [&>td:first-child]:before:w-1 [&>td:first-child]:before:bg-[#E4000F] [&>td:first-child]:before:transition-transform [&>td:first-child]:before:duration-200 [&>td:first-child]:before:origin-top [&>td:first-child]:before:scale-y-0 hover:[&>td:first-child]:before:scale-y-100">
						<td class="px-5 py-3 text-gray-500 text-xs tabular-nums">{formatDate(m.recorded_at)}</td>
						<td class="px-4 py-3 text-right font-semibold {wClass}">
							{m.weight_kg != null ? `${m.weight_kg}` : '—'}
							{#if m.weight_kg != null}<span class="text-xs font-normal text-gray-400 ml-0.5">kg</span>{/if}
						</td>
						<td class="px-4 py-3 text-right font-medium {fClass}">
							{m.body_fat_pct != null ? `${m.body_fat_pct}` : '—'}
							{#if m.body_fat_pct != null}<span class="text-xs font-normal text-gray-400">%</span>{/if}
						</td>
						<td class="px-4 py-3 text-right font-medium {mClass}">
							{m.muscle_pct != null ? `${m.muscle_pct}` : '—'}
							{#if m.muscle_pct != null}<span class="text-xs font-normal text-gray-400">%</span>{/if}
						</td>
						<td class="px-4 py-3 text-right text-sm font-medium {vClass}">
							{m.visceral_fat != null ? `Lv.${m.visceral_fat}` : '—'}
						</td>
						<td class="px-4 py-3 text-right">
							<div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-all duration-150">
								<button
									onclick={() => openEdit(m)}
									class="w-6 h-6 rounded flex items-center justify-center text-gray-300 hover:text-[#0EA5E9] hover:bg-sky-50 transition-colors text-sm"
									aria-label="編輯"
								>
									✎
								</button>
								<button
									onclick={() => (confirmDeleteId = m.id)}
									disabled={deleting.has(m.id)}
									class="w-6 h-6 rounded flex items-center justify-center text-gray-300 hover:text-red-500 hover:bg-red-50 transition-colors disabled:opacity-30"
									aria-label="刪除"
								>
									{deleting.has(m.id) ? '…' : '×'}
								</button>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	<!-- ── Mobile Cards (< md) ── -->
	<div class="md:hidden flex flex-col gap-3">
		{#each data.metrics as m, i (m.id)}
			{@const prev = data.metrics[i + 1]}
			{@const wClass = deltaClass(delta(m.weight_kg, prev?.weight_kg), false)}
			{@const fClass = deltaClass(delta(m.body_fat_pct, prev?.body_fat_pct), false)}
			{@const mClass = deltaClass(delta(m.muscle_pct, prev?.muscle_pct), true)}
			{@const vClass = deltaClass(delta(m.visceral_fat, prev?.visceral_fat), false)}
			<div class="bg-white rounded-xl shadow-sm overflow-hidden border-l-4 border-[#E4000F]">
				<div class="flex items-center justify-between px-4 pt-3 pb-2 border-b border-gray-50">
					<span class="text-xs text-gray-400 tabular-nums font-medium">{formatDate(m.recorded_at)}</span>
					<div class="flex items-center gap-1">
						<button
							onclick={() => openEdit(m)}
							class="w-7 h-7 rounded flex items-center justify-center text-gray-300 hover:text-[#0EA5E9] hover:bg-sky-50 transition-colors text-sm"
							aria-label="編輯"
						>
							✎
						</button>
						<button
							onclick={() => (confirmDeleteId = m.id)}
							disabled={deleting.has(m.id)}
							class="w-7 h-7 rounded flex items-center justify-center text-gray-300 hover:text-red-500 hover:bg-red-50 transition-colors disabled:opacity-30"
							aria-label="刪除"
						>
							{deleting.has(m.id) ? '…' : '×'}
						</button>
					</div>
				</div>
				<div class="grid grid-cols-2 gap-x-4 gap-y-2 px-4 py-3">
					<div>
						<p class="text-xs text-gray-400 mb-0.5">體重</p>
						<p class="font-semibold text-sm {wClass}">
							{m.weight_kg != null ? `${m.weight_kg}` : '—'}{#if m.weight_kg != null}<span class="text-xs font-normal text-gray-400 ml-0.5">kg</span>{/if}
						</p>
					</div>
					<div>
						<p class="text-xs text-gray-400 mb-0.5">體脂率</p>
						<p class="font-medium text-sm {fClass}">
							{m.body_fat_pct != null ? `${m.body_fat_pct}` : '—'}{#if m.body_fat_pct != null}<span class="text-xs font-normal text-gray-400">%</span>{/if}
						</p>
					</div>
					<div>
						<p class="text-xs text-gray-400 mb-0.5">肌肉率</p>
						<p class="font-medium text-sm {mClass}">
							{m.muscle_pct != null ? `${m.muscle_pct}` : '—'}{#if m.muscle_pct != null}<span class="text-xs font-normal text-gray-400">%</span>{/if}
						</p>
					</div>
					<div>
						<p class="text-xs text-gray-400 mb-0.5">內臟脂肪</p>
						<p class="text-sm {vClass}">
							{m.visceral_fat != null ? `Lv.${m.visceral_fat}` : '—'}
						</p>
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}

<!-- ── Delete Confirmation Modal ────────────────────── -->
{#if confirmDeleteId}
	{@const target = data.metrics.find((m) => m.id === confirmDeleteId)}
	<button
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 bg-black/60 z-50 w-full cursor-default"
		onclick={() => (confirmDeleteId = null)}
		aria-label="關閉對話框"
	></button>
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none">
		<div
			transition:fly={{ y: 16, duration: 200 }}
			class="bg-white rounded-xl p-6 max-w-sm w-full shadow-2xl pointer-events-auto"
			role="dialog"
			aria-modal="true"
			tabindex="-1"
		>
			<h3 class="font-bold text-gray-900 mb-1">確認刪除</h3>
			<p class="text-sm text-gray-500 mt-2 mb-5">
				確定要刪除 <span class="font-medium text-gray-700">{target ? formatDate(target.recorded_at) : ''}</span> 的紀錄？此操作無法復原。
			</p>
			<div class="flex justify-end gap-2">
				<button
					onclick={() => (confirmDeleteId = null)}
					class="px-4 py-2 text-sm text-gray-500 hover:text-gray-700 transition-colors"
				>
					取消
				</button>
				<button
					onclick={executeDelete}
					class="bg-red-500 text-white text-sm font-bold px-5 py-2 rounded-lg hover:bg-red-600 transition-colors"
				>
					確認刪除
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- ── Edit Modal ────────────────────────────────────── -->
{#if editingMetric}
	<button
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 bg-black/60 z-50 w-full cursor-default"
		onclick={() => (editingMetric = null)}
		aria-label="關閉對話框"
	></button>
	<div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 pointer-events-none">
		<div
			transition:fly={{ y: 16, duration: 200 }}
			class="bg-white rounded-xl w-full max-w-lg shadow-2xl pointer-events-auto"
			role="dialog"
			aria-modal="true"
			tabindex="-1"
		>
			<div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-gray-100">
				<span class="font-black tracking-widest text-sm text-gray-800">編輯記錄</span>
				<button
					onclick={() => (editingMetric = null)}
					class="text-gray-400 hover:text-gray-600 text-xl leading-none transition-colors"
					aria-label="關閉"
				>
					×
				</button>
			</div>
			<form onsubmit={handleEditSubmit} class="p-5 grid grid-cols-2 gap-4">
				<div class="col-span-2">
					<label for="e_recorded_at" class="block text-xs text-gray-400 mb-1">記錄時間 *</label>
					<input
						id="e_recorded_at"
						type="datetime-local"
						bind:value={e_recorded_at}
						required
						class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
					/>
				</div>
				<div>
					<label for="e_weight_kg" class="block text-xs text-gray-400 mb-1">體重 (kg)</label>
					<input
						id="e_weight_kg"
						type="number"
						step="0.1"
						min="30"
						max="300"
						bind:value={e_weight_kg}
						placeholder="72.5"
						class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
					/>
				</div>
				<div>
					<label for="e_body_fat_pct" class="block text-xs text-gray-400 mb-1">體脂率 (%)</label>
					<input
						id="e_body_fat_pct"
						type="number"
						step="0.1"
						min="1"
						max="70"
						bind:value={e_body_fat_pct}
						placeholder="18.2"
						class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
					/>
				</div>
				<div>
					<label for="e_muscle_pct" class="block text-xs text-gray-400 mb-1">肌肉率 (%)</label>
					<input
						id="e_muscle_pct"
						type="number"
						step="0.1"
						min="10"
						max="80"
						bind:value={e_muscle_pct}
						placeholder="35.2"
						class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
					/>
				</div>
				<div>
					<label for="e_visceral_fat" class="block text-xs text-gray-400 mb-1">內臟脂肪</label>
					<input
						id="e_visceral_fat"
						type="number"
						min="1"
						max="30"
						bind:value={e_visceral_fat}
						placeholder="8"
						class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]"
					/>
				</div>
				<div class="col-span-2">
					<label for="e_note" class="block text-xs text-gray-400 mb-1">備註（選填）</label>
					<textarea
						id="e_note"
						bind:value={e_note}
						rows="2"
						class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 resize-none focus:outline-none focus:border-[#E4000F]"
					></textarea>
				</div>
				<div class="col-span-2 flex justify-end gap-2 pt-1">
					<button
						type="button"
						onclick={() => (editingMetric = null)}
						class="px-4 py-2 text-xs text-gray-400 hover:text-gray-600 transition-colors"
					>
						取消
					</button>
					<button
						type="submit"
						disabled={editSubmitting}
						class="bg-gray-900 text-white text-xs font-bold px-5 py-2 rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-40"
					>
						{editSubmitting ? '儲存中…' : '儲存變更'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
