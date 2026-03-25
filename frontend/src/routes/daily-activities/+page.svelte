<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import { slide, fade, fly } from 'svelte/transition'
  import { createDailyActivity, updateDailyActivity, deleteDailyActivity } from '$lib/api/daily-activities'
  import type { DailyActivity, CommuteMode } from '$lib/types'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  let showForm = $state(false)
  let submitting = $state(false)
  let deleting = $state(new Set<string>())
  let confirmDeleteId = $state<string | null>(null)
  let editingActivity = $state<DailyActivity | null>(null)
  let editSubmitting = $state(false)

  // Form fields
  let activity_date = $state('')
  let steps = $state('')
  let commute_mode = $state('')
  let commute_minutes = $state('')
  let note = $state('')

  // Edit fields
  let e_steps = $state('')
  let e_commute_mode = $state('')
  let e_commute_minutes = $state('')
  let e_note = $state('')

  function today() {
    return new Date().toISOString().slice(0, 10)
  }

  function openForm() {
    activity_date = today()
    steps = ''
    commute_mode = ''
    commute_minutes = ''
    note = ''
    showForm = true
  }

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault()
    submitting = true
    try {
      await createDailyActivity({
        activity_date,
        ...(steps && { steps: parseInt(steps) }),
        ...(commute_mode && { commute_mode: commute_mode as CommuteMode }),
        ...(commute_minutes && { commute_minutes: parseInt(commute_minutes) }),
        ...(note && { note })
      })
      showForm = false
      await invalidateAll()
    } catch (err) {
      const msg = err instanceof Error ? err.message : '未知錯誤'
      if (msg.includes('already exists')) {
        alert('該日期已有紀錄，請直接編輯現有記錄')
      } else {
        alert('新增失敗：' + msg)
      }
    } finally {
      submitting = false
    }
  }

  function openEdit(activity: DailyActivity) {
    editingActivity = activity
    e_steps = activity.steps != null ? String(activity.steps) : ''
    e_commute_mode = activity.commute_mode ?? ''
    e_commute_minutes = activity.commute_minutes != null ? String(activity.commute_minutes) : ''
    e_note = activity.note ?? ''
  }

  async function handleEditSubmit(ev: SubmitEvent) {
    ev.preventDefault()
    if (!editingActivity) return
    editSubmitting = true
    try {
      await updateDailyActivity(editingActivity.id, {
        ...(e_steps && { steps: parseInt(e_steps) }),
        ...(e_commute_mode && { commute_mode: e_commute_mode as CommuteMode }),
        ...(e_commute_minutes && { commute_minutes: parseInt(e_commute_minutes) }),
        ...(e_note && { note: e_note })
      })
      editingActivity = null
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
      await deleteDailyActivity(id)
      await invalidateAll()
    } catch (err) {
      alert('刪除失敗：' + (err instanceof Error ? err.message : '未知錯誤'))
    } finally {
      deleting = new Set([...deleting].filter(d => d !== id))
    }
  }

  const commuteModeLabels: Record<string, string> = {
    scooter: '機車',
    train: '火車',
    walk: '步行',
    other: '其他'
  }
</script>

<!-- Header -->
<div class="flex items-center justify-between mb-5">
  <div>
    <h1 class="text-white font-black tracking-widest text-xl">每日活動</h1>
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
      <span class="font-black tracking-widest text-sm text-gray-800">新增每日活動</span>
    </div>
    <form onsubmit={handleSubmit} class="p-5 grid grid-cols-2 md:grid-cols-4 gap-4">
      <div class="col-span-2">
        <label for="activity_date" class="block text-xs text-gray-400 mb-1">日期 *</label>
        <input id="activity_date" type="date" bind:value={activity_date} required
          class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
      </div>
      <div>
        <label for="steps" class="block text-xs text-gray-400 mb-1">步數</label>
        <input id="steps" type="number" min="0" bind:value={steps} placeholder="8500"
          class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
      </div>
      <div>
        <label for="commute_mode" class="block text-xs text-gray-400 mb-1">通勤模式</label>
        <select id="commute_mode" bind:value={commute_mode}
          class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]">
          <option value="">—</option>
          <option value="scooter">機車</option>
          <option value="train">火車</option>
          <option value="walk">步行</option>
          <option value="other">其他</option>
        </select>
      </div>
      <div>
        <label for="commute_minutes" class="block text-xs text-gray-400 mb-1">通勤時長（分）</label>
        <input id="commute_minutes" type="number" min="0" bind:value={commute_minutes} placeholder="45"
          class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
      </div>
      <div class="col-span-2 md:col-span-3">
        <label for="note" class="block text-xs text-gray-400 mb-1">備註（選填）</label>
        <textarea id="note" bind:value={note} rows="2" placeholder="今天活動狀況…"
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
{#if data.activities.length === 0}
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
          <th class="text-left px-5 py-3 font-medium">日期</th>
          <th class="text-right px-4 py-3 font-medium">步數</th>
          <th class="text-left px-4 py-3 font-medium">通勤</th>
          <th class="text-right px-4 py-3 font-medium">通勤時長</th>
          <th class="w-20 px-4 py-3"></th>
        </tr>
      </thead>
      <tbody>
        {#each data.activities as activity (activity.id)}
          <tr class="border-b border-gray-50 last:border-0 hover:bg-gray-50 transition-colors group">
            <td class="px-5 py-3 text-gray-700 text-sm font-medium tabular-nums">{activity.activity_date}</td>
            <td class="px-4 py-3 text-right text-sm font-medium text-gray-700">
              {activity.steps != null ? activity.steps.toLocaleString() : '—'}
            </td>
            <td class="px-4 py-3 text-sm text-gray-500">
              {activity.commute_mode ? commuteModeLabels[activity.commute_mode] : '—'}
            </td>
            <td class="px-4 py-3 text-right text-sm text-gray-500">
              {activity.commute_minutes != null ? `${activity.commute_minutes} 分` : '—'}
            </td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-all duration-150">
                <button onclick={() => openEdit(activity)}
                  class="w-6 h-6 rounded flex items-center justify-center text-gray-300 hover:text-[#0EA5E9] hover:bg-sky-50 transition-colors text-sm"
                  aria-label="編輯">✎</button>
                <button onclick={() => (confirmDeleteId = activity.id)} disabled={deleting.has(activity.id)}
                  class="w-6 h-6 rounded flex items-center justify-center text-gray-300 hover:text-red-500 hover:bg-red-50 transition-colors disabled:opacity-30"
                  aria-label="刪除">{deleting.has(activity.id) ? '…' : '×'}</button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <!-- Mobile Cards -->
  <div class="md:hidden flex flex-col gap-3">
    {#each data.activities as activity (activity.id)}
      <div class="bg-white rounded-xl shadow-sm overflow-hidden border-l-4 border-gray-200">
        <div class="flex items-center justify-between px-4 pt-3 pb-2 border-b border-gray-50">
          <span class="text-sm font-medium text-gray-700 tabular-nums">{activity.activity_date}</span>
          <div class="flex items-center gap-1">
            <button onclick={() => openEdit(activity)}
              class="w-7 h-7 rounded flex items-center justify-center text-gray-300 hover:text-[#0EA5E9] transition-colors text-sm"
              aria-label="編輯">✎</button>
            <button onclick={() => (confirmDeleteId = activity.id)} disabled={deleting.has(activity.id)}
              class="w-7 h-7 rounded flex items-center justify-center text-gray-300 hover:text-red-500 transition-colors disabled:opacity-30"
              aria-label="刪除">{deleting.has(activity.id) ? '…' : '×'}</button>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-x-4 gap-y-2 px-4 py-3">
          <div>
            <p class="text-xs text-gray-400 mb-0.5">步數</p>
            <p class="font-medium text-sm text-gray-700">
              {activity.steps != null ? activity.steps.toLocaleString() : '—'}
            </p>
          </div>
          <div>
            <p class="text-xs text-gray-400 mb-0.5">通勤</p>
            <p class="text-sm text-gray-500">
              {activity.commute_mode ? commuteModeLabels[activity.commute_mode] : '—'}
              {activity.commute_minutes != null ? ` · ${activity.commute_minutes}m` : ''}
            </p>
          </div>
        </div>
      </div>
    {/each}
  </div>
{/if}

<!-- Delete Modal -->
{#if confirmDeleteId}
  {@const target = data.activities.find(a => a.id === confirmDeleteId)}
  <button transition:fade={{ duration: 150 }} class="fixed inset-0 bg-black/60 z-50 w-full cursor-default"
    onclick={() => (confirmDeleteId = null)} aria-label="關閉對話框"></button>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none">
    <div transition:fly={{ y: 16, duration: 200 }}
      class="bg-white rounded-xl p-6 max-w-sm w-full shadow-2xl pointer-events-auto"
      role="dialog" aria-modal="true">
      <h3 class="font-bold text-gray-900 mb-1">確認刪除</h3>
      <p class="text-sm text-gray-500 mt-2 mb-5">
        確定要刪除 <span class="font-medium text-gray-700">{target?.activity_date ?? ''}</span> 的紀錄？
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
{#if editingActivity}
  <button transition:fade={{ duration: 150 }} class="fixed inset-0 bg-black/60 z-50 w-full cursor-default"
    onclick={() => (editingActivity = null)} aria-label="關閉對話框"></button>
  <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 pointer-events-none">
    <div transition:fly={{ y: 16, duration: 200 }}
      class="bg-white rounded-xl w-full max-w-lg shadow-2xl pointer-events-auto"
      role="dialog" aria-modal="true">
      <div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-gray-100">
        <span class="font-black tracking-widest text-sm text-gray-800">編輯紀錄 — {editingActivity.activity_date}</span>
        <button onclick={() => (editingActivity = null)}
          class="text-gray-400 hover:text-gray-600 text-xl leading-none transition-colors" aria-label="關閉">×</button>
      </div>
      <form onsubmit={handleEditSubmit} class="p-5 grid grid-cols-2 gap-4">
        <div>
          <label for="e_steps" class="block text-xs text-gray-400 mb-1">步數</label>
          <input id="e_steps" type="number" min="0" bind:value={e_steps}
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
        </div>
        <div>
          <label for="e_commute_mode" class="block text-xs text-gray-400 mb-1">通勤模式</label>
          <select id="e_commute_mode" bind:value={e_commute_mode}
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]">
            <option value="">—</option>
            <option value="scooter">機車</option>
            <option value="train">火車</option>
            <option value="walk">步行</option>
            <option value="other">其他</option>
          </select>
        </div>
        <div>
          <label for="e_commute_minutes" class="block text-xs text-gray-400 mb-1">通勤時長（分）</label>
          <input id="e_commute_minutes" type="number" min="0" bind:value={e_commute_minutes}
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
        </div>
        <div>
          <label for="e_note" class="block text-xs text-gray-400 mb-1">備註</label>
          <input id="e_note" type="text" bind:value={e_note}
            class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-800 focus:outline-none focus:border-[#E4000F]" />
        </div>
        <div class="col-span-2 flex justify-end gap-2 pt-1">
          <button type="button" onclick={() => (editingActivity = null)}
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
