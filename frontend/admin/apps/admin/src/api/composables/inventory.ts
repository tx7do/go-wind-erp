import type {
  inventoryservicev1_DeleteInventoryRequest,
  inventoryservicev1_GetInventoryOverviewRequest,
  inventoryservicev1_GetInventoryRequest,
  inventoryservicev1_Inventory,
  inventoryservicev1_Inventory_Status,
  inventoryservicev1_InventoryOverview,
  inventoryservicev1_ListInventoryResponse,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 库存管理
// ==============================

export function useListInventories(
  query: PaginationQuery,
  options?: UseQueryOptions<inventoryservicev1_ListInventoryResponse, Error>,
) {
  return useQuery({
    queryKey: ['listInventories', query],
    queryFn: () => apiClient.inventoryService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListInventories(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listInventories', params],
    queryFn: () => apiClient.inventoryService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetInventory(
  req: inventoryservicev1_GetInventoryRequest,
  options?: UseQueryOptions<inventoryservicev1_Inventory, Error>,
) {
  return useQuery({
    queryKey: ['getInventory', req],
    queryFn: () => apiClient.inventoryService.Get(req),
    ...options,
  });
}

// ==============================
// 库存经营总览（看板聚合）
// ==============================

export function useGetInventoryOverview(
  req: inventoryservicev1_GetInventoryOverviewRequest,
  options?: UseQueryOptions<inventoryservicev1_InventoryOverview, Error>,
) {
  return useQuery({
    queryKey: ['getInventoryOverview', req],
    queryFn: () => apiClient.inventoryService.GetOverview(req),
    ...options,
  });
}

export async function fetchInventoryOverview(
  req: inventoryservicev1_GetInventoryOverviewRequest,
) {
  return queryClient.fetchQuery({
    queryKey: ['getInventoryOverview', req],
    queryFn: () => apiClient.inventoryService.GetOverview(req),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreateInventory(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.inventoryService.Create({
        data: { ...values } as inventoryservicev1_Inventory,
      }),
    ...options,
  });
}

export function useUpdateInventory(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.inventoryService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteInventory(
  options?: UseMutationOptions<
    object,
    Error,
    inventoryservicev1_DeleteInventoryRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.inventoryService.Delete(req),
    ...options,
  });
}

// ==============================
// 库存状态枚举与工具函数
// ==============================

export const inventoryStatusList = computed(() => [
  { value: 'AVAILABLE', label: t('enum.inventory.status.Available') },
  { value: 'LOCKED', label: t('enum.inventory.status.Locked') },
  { value: 'QUARANTINED', label: t('enum.inventory.status.Quarantined') },
]);

export function inventoryStatusToName(status: inventoryservicev1_Inventory_Status) {
  const values = inventoryStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function inventoryStatusToColor(
  status: inventoryservicev1_Inventory_Status,
) {
  switch (status) {
    case 'AVAILABLE': {
      return 'green';
    }
    case 'LOCKED': {
      return 'orange';
    }
    case 'QUARANTINED': {
      return 'red';
    }
    default: {
      return 'gray';
    }
  }
}
