import { PUBLIC_API_BASE_URL } from '$env/static/public';

interface ApiError {
  code: string;
  message: string;
  details?: { field: string; issue: string }[];
}

export class ApiException extends Error {
  constructor(
    public status: number,
    public error: ApiError
  ) {
    super(error.message);
  }
}

function makeRequest(fetchFn: typeof fetch) {
  return async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetchFn(`${PUBLIC_API_BASE_URL}${path}`, {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      ...init
    });

    if (res.status === 401 && path !== '/auth/refresh') {
      const refreshRes = await fetchFn(`${PUBLIC_API_BASE_URL}/auth/refresh`, {
        method: 'POST',
        credentials: 'include'
      });
      if (refreshRes.ok) {
        const retryRes = await fetchFn(`${PUBLIC_API_BASE_URL}${path}`, {
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          ...init
        });
        if (retryRes.ok) {
          if (retryRes.status === 204) return undefined as T;
          return retryRes.json();
        }
        const retryBody = await retryRes.json();
        throw new ApiException(retryRes.status, retryBody.error);
      }
    }

    if (!res.ok) {
      const body = await res.json();
      throw new ApiException(res.status, body.error);
    }

    if (res.status === 204) return undefined as T;
    return res.json();
  };
}

export function createApi(fetchFn: typeof fetch = fetch) {
  const request = makeRequest(fetchFn);
  return {
    get: <T>(path: string) => request<T>(path),
    post: <T>(path: string, body: unknown) =>
      request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
    patch: <T>(path: string, body: unknown) =>
      request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
    delete: (path: string) => request<void>(path, { method: 'DELETE' })
  };
}

export const api = createApi();
