<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';
	import { authStore, isLoading } from '$lib/stores/auth';

	let { children } = $props();

	const PUBLIC_ROUTES = ['/login', '/auth/callback'];

	onMount(async () => {
		await authStore.init();
	});

	$effect(() => {
		const currentPath = $page.url.pathname;
		const isPublic = PUBLIC_ROUTES.some((r) => currentPath.startsWith(r));
		if (!$isLoading && !$authStore && !isPublic) {
			goto('/login');
		}
	});

	const navItems = [
		{ label: '體位數據', href: '/body-metrics', enabled: true },
		{ label: '睡眠', href: '/sleep-logs', enabled: true },
		{ label: '活動', href: '/daily-activities', enabled: true },
	];

	async function handleLogout() {
		await authStore.logout();
		goto('/login');
	}
</script>

<svelte:head>
	<title>健康追蹤計劃</title>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-screen bg-[#1a1a2e]">
	<nav class="h-14 flex items-center px-6 bg-[#E4000F]">
		<a href="/" class="text-white font-black tracking-widest text-lg mr-auto hover:opacity-80 transition-opacity">HEALTH TRACKER</a>
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
	<main class="max-w-5xl mx-auto px-6 py-8">
		{@render children()}
	</main>
</div>
