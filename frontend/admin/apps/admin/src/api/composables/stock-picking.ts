import type {
  inventoryservicev1_CreateStockPickingRequest,
  inventoryservicev1_DeleteStockPickingRequest,
  inventoryservicev1_GetStockPickingRequest,
  inventoryservicev1_ListStockPickingResponse,
  inventoryservicev1_StockPicking,
  inventoryservicev1_StockPicking_DerivedState,
  inventoryservicev1_StockPicking_PickingType,
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
// 拣货单（借鉴 Odoo stock.picking：一等文档，生命周期从子 moves 派生）
// ==============================

export function useListStockPickings(
  query: PaginationQuery,
  options?: UseQueryOptions<
    inventoryservicev1_ListStockPickingResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listStockPickings', query],
    queryFn: () => apiClient.stockPickingService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListStockPickings(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listStockPickings', params],
    queryFn: () => apiClient.stockPickingService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetStockPicking(
  req: inventoryservicev1_GetStockPickingRequest,
  options?: UseQueryOptions<inventoryservicev1_StockPicking, Error>,
) {
  return useQuery({
    queryKey: ['getStockPicking', req],
    queryFn: () => apiClient.stockPickingService.Get(req),
    ...options,
  });
}

export function useCreateStockPicking(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.stockPickingService.Create({
        data: { ...values } as inventoryservicev1_StockPicking,
      } as inventoryservicev1_CreateStockPickingRequest),
    ...options,
  });
}

export function useConfirmStockPicking(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }: { id: number }) =>
      apiClient.stockPickingService.Confirm({ id }),
    ...options,
  });
}

export function useValidateStockPicking(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }: { id: number }) =>
      apiClient.stockPickingService.Validate({ id }),
    ...options,
  });
}

export function useCancelStockPicking(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }: { id: number }) =>
      apiClient.stockPickingService.Cancel({ id }),
    ...options,
  });
}

export function useDeleteStockPicking(
  options?: UseMutationOptions<
    object,
    Error,
    inventoryservicev1_DeleteStockPickingRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.stockPickingService.Delete(req),
    ...options,
  });
}

// ==============================
// 拣货类型 / 派生态 枚举与工具函数
// ==============================

export const pickingTypeList = computed(() => [
  { value: 'INCOMING', label: t('enum.stockPicking.pickingType.Incoming') },
  { value: 'INTERNAL', label: t('enum.stockPicking.pickingType.Internal') },
]);

export function pickingTypeToName(
  type: inventoryservicev1_StockPicking_PickingType,
) {
  const values = pickingTypeList.value;
  const matchedItem = values.find((item) => item.value === type);
  return matchedItem ? matchedItem.label : '';
}

export function pickingTypeToColor(
  type: inventoryservicev1_StockPicking_PickingType,
) {
  switch (type) {
    case 'INCOMING': {
      return 'green';
    }
    case 'INTERNAL': {
      return 'blue';
    }
    default: {
      return 'gray';
    }
  }
}

export const derivedStateList = computed(() => [
  { value: 'DRAFT', label: t('enum.stockPicking.derivedState.Draft') },
  { value: 'CONFIRMED', label: t('enum.stockPicking.derivedState.Confirmed') },
  { value: 'DONE', label: t('enum.stockPicking.derivedState.Done') },
  { value: 'CANCELLED', label: t('enum.stockPicking.derivedState.Cancelled') },
]);

export function derivedStateToName(
  state: inventoryservicev1_StockPicking_DerivedState,
) {
  const values = derivedStateList.value;
  const matchedItem = values.find((item) => item.value === state);
  return matchedItem ? matchedItem.label : '';
}

export function derivedStateToColor(
  state: inventoryservicev1_StockPicking_DerivedState,
) {
  switch (state) {
    case 'DRAFT': {
      return 'gray';
    }
    case 'CONFIRMED': {
      return 'orange';
    }
    case 'DONE': {
      return 'green';
    }
    case 'CANCELLED': {
      return 'red';
    }
    default: {
      return 'gray';
    }
  }
}
