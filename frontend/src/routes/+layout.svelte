<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';
	import { authStore, isLoading } from '$lib/stores/auth';

	let { children } = $props();

	const PUBLIC_ROUTES = [resolve('/login'), resolve('/auth/callback')];

	const isPublicRoute = $derived(
		PUBLIC_ROUTES.some((r) => page.url.pathname.startsWith(r))
	);

	$effect(() => {
		authStore.init();
	});

	$effect(() => {
		const currentPath = page.url.pathname;
		const isPublic = PUBLIC_ROUTES.some((r) => currentPath.startsWith(r));
		if (!$isLoading && !$authStore && !isPublic) {
			goto(resolve('/login'));
		}
	});

	const navItems = [
		{ label: '體位數據', href: resolve('/body-metrics'), enabled: true },
		{ label: '睡眠', href: resolve('/sleep-logs'), enabled: true },
		{ label: '活動', href: resolve('/daily-activities'), enabled: true },
	];

	async function handleLogout() {
		await authStore.logout();
		goto(resolve('/login'));
	}
</script>

<svelte:head>
	<title>健康追蹤計劃</title>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-screen bg-[#1a1a2e]">
	{#if !isPublicRoute}
	<nav class="h-14 flex items-center px-6 bg-[#E4000F]">
		<a href={resolve('/')} class="text-white font-black tracking-widest text-lg mr-auto hover:opacity-80 transition-opacity">HEALTH TRACKER</a>
		<div class="flex items-center gap-6">
			{#each navItems as item}
				{#if item.enabled}
					<a href={item.href} class="text-white font-bold text-sm tracking-wide hover:opacity-80 transition-opacity">
						{item.label}
					</a>
				{:else}
					<span class="text-white font-bold text-sm tracking-wide opacity-40 cursor-not-allowed">
						{item.label}
					</span>
				{/if}
			{/each}
			{#if $authStore}
				<button
					onclick={handleLogout}
					class="text-white font-bold text-sm tracking-wide hover:opacity-80 transition-opacity"
				>
					登出
				</button>
			{/if}
		</div>
	</nav>
	{/if}
	{#if isPublicRoute}
		{@render children()}
	{:else}
		<main class="max-w-5xl mx-auto px-6 py-8">
			{@render children()}
		</main>
	{/if}
</div>
