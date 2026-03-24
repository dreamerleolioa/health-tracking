<script lang="ts">
	const today = new Date().toLocaleDateString('zh-TW', {
		year: 'numeric',
		month: 'long',
		day: 'numeric',
		weekday: 'long',
	});

	// Mock 數值，之後接 API 替換
	const metrics = [
		{
			label: '體重',
			emoji: '⚖️',
			prefix: '',
			value: '72.5',
			unit: 'kg',
			color: '#0EA5E9',
		},
		{
			label: '體脂率',
			emoji: '🔥',
			prefix: '',
			value: '18.2',
			unit: '%',
			color: '#F59E0B',
		},
		{
			label: '肌肉率',
			emoji: '💪',
			prefix: '',
			value: '35.2',
			unit: '%',
			color: '#10B981',
		},
		{
			label: '內臟脂肪',
			emoji: '📊',
			prefix: 'Lv.',
			value: '8',
			unit: '',
			color: '#8B5CF6',
		},
	];
</script>

<!-- 頁面標題 -->
<div class="mb-8">
	<h1 class="text-white font-black tracking-widest text-2xl">TODAY</h1>
	<p class="text-gray-400 text-sm mt-1">{today}</p>
</div>

<!-- 數據卡片 -->
<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
	{#each metrics as metric}
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
