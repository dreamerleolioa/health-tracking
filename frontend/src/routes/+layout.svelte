<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';

	let { children } = $props();

	const navItems = [
		{ label: '體位數據', href: '/body-metrics', enabled: true },
		{ label: '睡眠', href: '/sleep', enabled: false },
		{ label: '活動', href: '/activities', enabled: false },
	];
</script>

<svelte:head>
	<title>健康追蹤計劃</title>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-screen bg-[#1a1a2e]">
	<!-- Top Nav -->
	<nav class="h-14 flex items-center px-6 bg-[#E4000F]">
		<span class="text-white font-black tracking-widest text-lg mr-auto">HEALTH TRACKER</span>
		<div class="flex items-center gap-6">
			{#each navItems as item}
				{#if item.enabled}
					<a
						href={item.href}
						class="text-white font-bold text-sm tracking-wide hover:opacity-80 transition-opacity"
					>
						{item.label}
					</a>
				{:else}
					<span class="text-white font-bold text-sm tracking-wide opacity-40 cursor-not-allowed">
						{item.label}
					</span>
				{/if}
			{/each}
		</div>
	</nav>

	<!-- Main Content -->
	<main class="max-w-5xl mx-auto px-6 py-8">
		{@render children()}
	</main>
</div>
