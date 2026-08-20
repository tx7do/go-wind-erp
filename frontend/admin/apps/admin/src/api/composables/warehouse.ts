import type {
  inventoryservicev1_DeleteWarehouseRequest,
  inventoryservicev1_GetWarehouseRequest,
  inventoryservicev1_ListWarehouseResponse,
  inventoryservicev1_Warehouse,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

// ==============================
// 仓库管理
// ==============================

export function useListWarehouses(
  query: PaginationQuery,
  options?: UseQueryOptions<inventoryservicev1_ListWarehouseResponse, Error>,
) {
  return useQuery({
    queryKey: ['listWarehouses', query],
    queryFn: () => apiClient.warehouseService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListWarehouses(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listWarehouses', params],
    queryFn: () => apiClient.warehouseService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetWarehouse(
  req: inventoryservicev1_GetWarehouseRequest,
  options?: UseQueryOptions<inventoryservicev1_Warehouse, Error>,
) {
  return useQuery({
    queryKey: ['getWarehouse', req],
    queryFn: () => apiClient.warehouseService.Get(req),
    ...options,
  });
}

export function useCreateWarehouse(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.warehouseService.Create({
        data: { ...values } as inventoryservicev1_Warehouse,
      }),
    ...options,
  });
}

export function useUpdateWarehouse(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.warehouseService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteWarehouse(
  options?: UseMutationOptions<
    object,
    Error,
    inventoryservicev1_DeleteWarehouseRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.warehouseService.Delete(req),
    ...options,
  });
}
