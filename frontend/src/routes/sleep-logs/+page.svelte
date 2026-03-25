<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import { slide, fade, fly } from 'svelte/transition'
  import { createSleepLog, updateSleepLog, deleteSleepLog } from '$lib/api/sleep-logs'
  import type { SleepLog } from '$lib/types'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  let showForm = $state(false)
  let submitting = $state(false)
  let deleting = $state(new Set<string>())
  let confirmDeleteId = $state<string | null>(null)
  let editingLog = $state<SleepLog | null>(null)
  let editSubmitting = $state(false)

  // Form fields
  let sleep_at = $state('')
  let wake_at = $state('')
  let quality = $state('')
  let note = $state('')

  // Edit fields
  let e_sleep_at = $state('')
  let e_wake_at = $state('')
  let e_quality = $state('')
  let e_note = $state('')

  function defaultDateTimeLocal(offsetHours = 0) {
    const now = new Date()
    now.setHours(now.getHours() + offsetHours)
    return new Date(now.getTime() - now.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
  }

  function openForm() {
    sleep_at = defaultDateTimeLocal(-8)
    wake_at = defaultDateTimeLocal(0)
    quality = ''
    note = ''
    showForm = true
  }

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault()
    submitting = true
    try {
      await createSleepLog({
        sleep_at: new Date(sleep_at).toISOString(),
        wake_at: new Date(wake_at).toISOString(),
        ...(quality && { quality: parseInt(quality) }),
        ...(note && { note })
      })
      showForm = false
      await invalidateAll()
    } catch (err) {
      alert('新增失敗：' + (err instanceof Error ? err.message : '未知錯誤'))
    } finally {
      submitting = false
    }
  }

  function openEdit(log: SleepLog) {
    editingLog = log
    e_sleep_at = toLocalInput(log.sleep_at)
    e_wake_at = toLocalInput(log.wake_at)
    e_quality = log.quality != null ? String(log.quality) : ''
    e_note = log.note ?? ''
  }

  async function handleEditSubmit(ev: SubmitEvent) {
    ev.preventDefault()
    if (!editingLog) return
    editSubmitting = true
    try {
      await updateSleepLog(editingLog.id, {
        sleep_at: new Date(e_sleep_at).toISOString(),
        wake_at: new Date(e_wake_at).toISOString(),
        ...(e_quality && { quality: parseInt(e_quality) }),
        ...(e_note && { note: e_note })
      })
      editingLog = null
      await invalidateAll()
    } catch (err) {
      alert('編輯失敗：' + (err instanceof Error ? err.message : '未知錯誤'))
    } finally {
      editSubmitting = false
    }
  }

  async function executeDelete() {
    if (!confirmDeleteId) return
    const id = confirmDeleteId
    confirmDeleteId = null
    deleting = new Set([...deleting, id])
    try {
      await deleteSleepLog(id)
      await invalidateAll()
    } catch (err) {
      alert('刪除失敗：' + (err instanceof Error ? err.message : '未知錯誤'))
    } finally {
      deleting = new Set([...deleting].filter(d => d !== id))
    }
  }

  function toLocalInput(isoStr: string) {
    const d = new Date(isoStr)
    return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
  }

  function formatDateTime(isoStr: string) {
    const d = new Date(isoStr)
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    const hh = String(d.getHours()).padStart(2, '0')
    const min = String(d.getMinutes()).padStart(2, '0')
    return `${mm}/${dd} ${hh}:${min}`
  }

  function formatDuration(minutes: number | null) {
    if (minutes == null) return '—'
    const h = Math.floor(minutes / 60)
    const m = minutes % 60
    return h > 0 ? `${h}h ${m}m` : `${m}m`
  }
</script>

<!-- Header -->
<div class="flex items-center justify-between mb-5">
  <div>
    <h1 class="text-white font-black tracking-widest text-xl">睡眠紀錄</h1>
    <p class="text-white/40 text-xs mt-0.5">共 {data.meta.total} 筆記錄</p>
  </div>
  <button
    onclick={openForm}
    class="bg-white text-gray-900 text-sm font-bold px-4 py-2 rounded-lg hover:bg-gray-100 transition-colors"
  >
    + 新增記錄
  </button>
</div>

<!-- Inline Form -->
{#if showForm}
  <div transition:slide={{ duration: 180 }} class="mb-5 bg-white rounded-xl shadow-sm">
    <div class="px-5 pt-4 pb-3 border-b border-gray-100">
      <span class="font-black tracking-widest text-sm text-gray-800">新增睡眠紀錄</span>
    </div>
    <form onsubmit={handleSubmit} class="p-5 grid grid-cols-2 md:grid-cols-4 gap-4">
      <div class="col-span-2">
        <label for="sleep_at" class="block text-xs text-gray-400 mb-1">上床時間 *</label>
        <input id="sleep_at" type="datetime-local" bind:value={sleep_at} required
          class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
      </div>
      <div class="col-span-2">
        <label for="wake_at" class="block text-xs text-gray-400 mb-1">起床時間 *</label>
        <input id="wake_at" type="datetime-local" bind:value={wake_at} required
          class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
      </div>
      <div>
        <label for="quality" class="block text-xs text-gray-400 mb-1">睡眠品質（1–5）</label>
        <input id="quality" type="number" min="1" max="5" bind:value={quality} placeholder="3"
          class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
      </div>
      <div class="col-span-2 md:col-span-3">
        <label for="note" class="block text-xs text-gray-400 mb-1">備註（選填）</label>
        <textarea id="note" bind:value={note} rows="2" placeholder="今天睡眠狀況…"
          class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 resize-none focus:outline-none focus:border-[#E4000F]"></textarea>
      </div>
      <div class="col-span-2 md:col-span-4 flex justify-end gap-2 pt-1">
        <button type="button" onclick={() => (showForm = false)}
          class="px-4 py-2 text-xs text-gray-400 hover:text-gray-600 transition-colors">取消</button>
        <button type="submit" disabled={submitting}
          class="bg-gray-900 text-white text-xs font-bold px-5 py-2 rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-40">
          {submitting ? '儲存中…' : '儲存'}
        </button>
      </div>
    </form>
  </div>
{/if}

<!-- Table / Empty State -->
{#if data.logs.length === 0}
  <div class="flex flex-col items-center justify-center py-20 text-white/30 gap-1">
    <p class="text-sm">尚無紀錄</p>
    <p class="text-xs">點擊「+ 新增記錄」開始追蹤</p>
  </div>
{:else}
  <!-- Desktop Table -->
  <div class="hidden md:block bg-white rounded-xl shadow-sm overflow-hidden">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wider">
          <th class="text-left px-5 py-3 font-medium">上床</th>
          <th class="text-left px-4 py-3 font-medium">起床</th>
          <th class="text-right px-4 py-3 font-medium">時長</th>
          <th class="text-right px-4 py-3 font-medium">品質</th>
          <th class="text-center px-4 py-3 font-medium">異常</th>
          <th class="w-20 px-4 py-3"></th>
        </tr>
      </thead>
      <tbody>
        {#each data.logs as log (log.id)}
          <tr class="border-b border-gray-50 last:border-0 hover:bg-gray-50 transition-colors group">
            <td class="px-5 py-3 text-gray-500 text-xs tabular-nums">{formatDateTime(log.sleep_at)}</td>
            <td class="px-4 py-3 text-gray-500 text-xs tabular-nums">{formatDateTime(log.wake_at)}</td>
            <td class="px-4 py-3 text-right text-sm font-medium text-gray-700">{formatDuration(log.duration_min)}</td>
            <td class="px-4 py-3 text-right text-sm">
              {log.quality != null ? '★'.repeat(log.quality) : '—'}
            </td>
            <td class="px-4 py-3 text-center text-sm">
              {#if log.abnormal_wake}
                <span class="text-orange-500 font-bold" title="異常喚醒（凌晨 3–4 點）">▲</span>
              {:else}
                <span class="text-gray-200">—</span>
              {/if}
            </td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-all duration-150">
                <button onclick={() => openEdit(log)}
                  class="w-6 h-6 rounded flex items-center justify-center text-gray-300 hover:text-[#0EA5E9] hover:bg-sky-50 transition-colors text-sm"
                  aria-label="編輯">✎</button>
                <button onclick={() => (confirmDeleteId = log.id)} disabled={deleting.has(log.id)}
                  class="w-6 h-6 rounded flex items-center justify-center text-gray-300 hover:text-red-500 hover:bg-red-50 transition-colors disabled:opacity-30"
                  aria-label="刪除">{deleting.has(log.id) ? '…' : '×'}</button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <!-- Mobile Cards -->
  <div class="md:hidden flex flex-col gap-3">
    {#each data.logs as log (log.id)}
      <div class="bg-white rounded-xl shadow-sm overflow-hidden"
           class:border-l-4={true}
           class:border-orange-400={log.abnormal_wake}
           class:border-gray-200={!log.abnormal_wake}>
        <div class="flex items-center justify-between px-4 pt-3 pb-2 border-b border-gray-50">
          <span class="text-xs text-gray-400 tabular-nums font-medium">{formatDateTime(log.sleep_at)} → {formatDateTime(log.wake_at)}</span>
          <div class="flex items-center gap-1">
            {#if log.abnormal_wake}
              <span class="text-orange-500 text-xs font-bold mr-1" title="異常喚醒">▲</span>
            {/if}
            <button onclick={() => openEdit(log)}
              class="w-7 h-7 rounded flex items-center justify-center text-gray-300 hover:text-[#0EA5E9] transition-colors text-sm"
              aria-label="編輯">✎</button>
            <button onclick={() => (confirmDeleteId = log.id)} disabled={deleting.has(log.id)}
              class="w-7 h-7 rounded flex items-center justify-center text-gray-300 hover:text-red-500 transition-colors disabled:opacity-30"
              aria-label="刪除">{deleting.has(log.id) ? '…' : '×'}</button>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-x-4 gap-y-2 px-4 py-3">
          <div>
            <p class="text-xs text-gray-400 mb-0.5">睡眠時長</p>
            <p class="font-medium text-sm text-gray-700">{formatDuration(log.duration_min)}</p>
          </div>
          <div>
            <p class="text-xs text-gray-400 mb-0.5">品質評分</p>
            <p class="text-sm">{log.quality != null ? '★'.repeat(log.quality) : '—'}</p>
          </div>
        </div>
      </div>
    {/each}
  </div>
{/if}

<!-- Delete Modal -->
{#if confirmDeleteId}
  {@const target = data.logs.find(l => l.id === confirmDeleteId)}
  <button transition:fade={{ duration: 150 }} class="fixed inset-0 bg-black/60 z-50 w-full cursor-default"
    onclick={() => (confirmDeleteId = null)} aria-label="關閉對話框"></button>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none">
    <div transition:fly={{ y: 16, duration: 200 }}
      class="bg-white rounded-xl p-6 max-w-sm w-full shadow-2xl pointer-events-auto"
      role="dialog" aria-modal="true">
      <h3 class="font-bold text-gray-900 mb-1">確認刪除</h3>
      <p class="text-sm text-gray-500 mt-2 mb-5">
        確定要刪除 <span class="font-medium text-gray-700">{target ? formatDateTime(target.sleep_at) : ''}</span> 的紀錄？
      </p>
      <div class="flex justify-end gap-2">
        <button onclick={() => (confirmDeleteId = null)}
          class="px-4 py-2 text-sm text-gray-500 hover:text-gray-700 transition-colors">取消</button>
        <button onclick={executeDelete}
          class="bg-red-500 text-white text-sm font-bold px-5 py-2 rounded-lg hover:bg-red-600 transition-colors">確認刪除</button>
      </div>
    </div>
  </div>
{/if}

<!-- Edit Modal -->
{#if editingLog}
  <button transition:fade={{ duration: 150 }} class="fixed inset-0 bg-black/60 z-50 w-full cursor-default"
    onclick={() => (editingLog = null)} aria-label="關閉對話框"></button>
  <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 pointer-events-none">
    <div transition:fly={{ y: 16, duration: 200 }}
      class="bg-white rounded-xl w-full max-w-lg shadow-2xl pointer-events-auto"
      role="dialog" aria-modal="true">
      <div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-gray-100">
        <span class="font-black tracking-widest text-sm text-gray-800">編輯紀錄</span>
        <button onclick={() => (editingLog = null)}
          class="text-gray-400 hover:text-gray-600 text-xl leading-none transition-colors" aria-label="關閉">×</button>
      </div>
      <form onsubmit={handleEditSubmit} class="p-5 grid grid-cols-2 gap-4">
        <div>
          <label for="e_sleep_at" class="block text-xs text-gray-400 mb-1">上床時間 *</label>
          <input id="e_sleep_at" type="datetime-local" bind:value={e_sleep_at} required
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
        </div>
        <div>
          <label for="e_wake_at" class="block text-xs text-gray-400 mb-1">起床時間 *</label>
          <input id="e_wake_at" type="datetime-local" bind:value={e_wake_at} required
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
        </div>
        <div>
          <label for="e_quality" class="block text-xs text-gray-400 mb-1">品質（1–5）</label>
          <input id="e_quality" type="number" min="1" max="5" bind:value={e_quality}
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
        </div>
        <div>
          <label for="e_note" class="block text-xs text-gray-400 mb-1">備註</label>
          <input id="e_note" type="text" bind:value={e_note}
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
        </div>
        <div class="col-span-2 flex justify-end gap-2 pt-1">
          <button type="button" onclick={() => (editingLog = null)}
            class="px-4 py-2 text-xs text-gray-400 hover:text-gray-600 transition-colors">取消</button>
          <button type="submit" disabled={editSubmitting}
            class="bg-gray-900 text-white text-xs font-bold px-5 py-2 rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-40">
            {editSubmitting ? '儲存中…' : '儲存變更'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
