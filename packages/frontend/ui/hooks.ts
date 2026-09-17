/**
 * RINCO shared React hooks:
 *   - useApiQuery: TanStack Query wrapper với error/loading defaults
 *   - useApiMutation: TanStack Query mutation wrapper
 *
 * Import từ frontend app nào cũng được. Yêu cầu @tanstack/react-query
 * đã được setup ở QueryClientProvider của app.
 */

import {
  useQuery,
  useMutation,
  type UseQueryOptions,
  type UseMutationOptions,
  type QueryKey,
} from "@tanstack/react-query";
import { apiFetch, ApiError, type FetchOptions } from "./api-client";

/**
 * Wrap một GET call với TanStack Query.
 *
 * Ví dụ:
 *   const { data, isLoading, error } = useApiQuery(
 *     ["tenants", page],
 *     () => apiGet<...>("/v1/tenants", { service: "tenant", headers: { ... } }),
 *     { staleTime: 30_000 }
 *   );
 */
export function useApiQuery<TData = unknown, TError = ApiError>(
  queryKey: QueryKey,
  queryFn: () => Promise<TData>,
  options?: Omit<UseQueryOptions<TData, TError, TData>, "queryKey" | "queryFn">,
) {
  return useQuery<TData, TError>({
    queryKey,
    queryFn,
    ...options,
  });
}

/**
 * Mutation hook với error type mặc định là ApiError.
 */
export function useApiMutation<TData = unknown, TVariables = unknown, TError = ApiError>(
  mutationFn: (variables: TVariables, options?: FetchOptions) => Promise<TData>,
  options?: UseMutationOptions<TData, TError, TVariables>,
) {
  return useMutation<TData, TError, TVariables>({
    mutationFn: (variables) => mutationFn(variables),
    ...options,
  });
}

/** Re-export để app code không cần import từ cả 2 file. */
export { ApiError, apiFetch };
