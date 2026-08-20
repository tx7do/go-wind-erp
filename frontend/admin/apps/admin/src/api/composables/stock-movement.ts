import type {
  inventoryservicev1_DeleteStockMovementRequest,
  inventoryservicev1_GetStockMovementRequest,
  inventoryservicev1_ListStockMovementResponse,
  inventoryservicev1_StockMovement,
  inventoryservicev1_StockMovement_MovementType,
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
import { type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 库存流水管理（追加型，仅创建/删除，无更新）
// ==============================

export function useListStockMovements(
  query: PaginationQuery,
  options?: UseQueryOptions<
    inventoryservicev1_ListStockMovementResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listStockMovements', query],
    queryFn: () => apiClient.stockMovementService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListStockMovements(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listStockMovements', params],
    queryFn: () => apiClient.stockMovementService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetStockMovement(
  req: inventoryservicev1_GetStockMovementRequest,
  options?: UseQueryOptions<inventoryservicev1_StockMovement, Error>,
) {
  return useQuery({
    queryKey: ['getStockMovement', req],
    queryFn: () => apiClient.stockMovementService.Get(req),
    ...options,
  });
}

export function useCreateStockMovement(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.stockMovementService.Create({
        data: { ...values } as inventoryservicev1_StockMovement,
      }),
    ...options,
  });
}

export function useDeleteStockMovement(
  options?: UseMutationOptions<
    object,
    Error,
    inventoryservicev1_DeleteStockMovementRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.stockMovementService.Delete(req),
    ...options,
  });
}

// ==============================
// 流水类型枚举与工具函数
// ==============================

export const movementTypeList = computed(() => [
  { value: 'INBOUND', label: t('enum.stockMovement.movementType.Inbound') },
  { value: 'OUTBOUND', label: t('enum.stockMovement.movementType.Outbound') },
  { value: 'TRANSFER', label: t('enum.stockMovement.movementType.Transfer') },
  {
    value: 'ADJUSTMENT',
    label: t('enum.stockMovement.movementType.Adjustment'),
  },
]);

export function movementTypeToName(
  type: inventoryservicev1_StockMovement_MovementType,
) {
  const values = movementTypeList.value;
  const matchedItem = values.find((item) => item.value === type);
  return matchedItem ? matchedItem.label : '';
}

export function movementTypeToColor(
  type: inventoryservicev1_StockMovement_MovementType,
) {
  switch (type) {
    case 'INBOUND': {
      return 'green';
    }
    case 'OUTBOUND': {
      return 'red';
    }
    case 'TRANSFER': {
      return 'blue';
    }
    case 'ADJUSTMENT': {
      return 'orange';
    }
    default: {
      return 'gray';
    }
  }
}
