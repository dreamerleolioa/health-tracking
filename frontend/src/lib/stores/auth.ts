import { writable, derived } from 'svelte/store';
import type { User, ItemResponse } from '$lib/types';
import { api } from '$lib/api/client';

function createAuthStore() {
  const user = writable<User | null | undefined>(undefined); // undefined = not yet loaded

  async function init() {
    try {
      const data = await api.get<ItemResponse<User>>('/auth/me');
      user.set(data.data);
    } catch {
      user.set(null);
    }
  }

  async function logout() {
    try {
      await api.post('/auth/logout', {});
    } finally {
      user.set(null);
    }
  }

  return { subscribe: user.subscribe, init, logout };
}

export const authStore = createAuthStore();
export const isLoggedIn = derived(authStore, ($user) => $user !== null && $user !== undefined);
export const isLoading = derived(authStore, ($user) => $user === undefined);
