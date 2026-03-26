<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { authStore } from '$lib/stores/auth';

  let error = $state('');

  onMount(async () => {
    try {
      await authStore.init();
      goto(`${base}/`);
    } catch {
      error = '登入失敗，請稍後再試。';
    }
  });
</script>

<div class="min-h-screen bg-[#1a1a2e] flex items-center justify-center">
  {#if error}
    <p class="text-red-400">{error}</p>
  {:else}
    <p class="text-gray-400">登入中，請稍候…</p>
  {/if}
</div>
