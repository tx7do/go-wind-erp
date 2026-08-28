import type {
  financeservicev1_CancelReceivableRequest,
  financeservicev1_DeleteReceivableRequest,
  financeservicev1_GetReceivableRequest,
  financeservicev1_ListReceivableResponse,
  financeservicev1_Receivable,
  financeservicev1_Receivable_Status,
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
// 应收单管理
// ==============================

export function useListReceivables(
  query: PaginationQuery,
  options?: UseQueryOptions<financeservicev1_ListReceivableResponse, Error>,
) {
  return useQuery({
    queryKey: ['listReceivables', query],
    queryFn: () => apiClient.receivableService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListReceivables(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listReceivables', params],
    queryFn: () => apiClient.receivableService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetReceivable(
  req: financeservicev1_GetReceivableRequest,
  options?: UseQueryOptions<financeservicev1_Receivable, Error>,
) {
  return useQuery({
    queryKey: ['getReceivable', req],
    queryFn: () => apiClient.receivableService.Get(req),
    ...options,
  });
}

export function useCreateReceivable(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.receivableService.Create({
        data: { ...values } as financeservicev1_Receivable,
      }),
    ...options,
  });
}

export function useDeleteReceivable(
  options?: UseMutationOptions<
    object,
    Error,
    financeservicev1_DeleteReceivableRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.receivableService.Delete(req),
    ...options,
  });
}

export function useCancelReceivable(
  options?: UseMutationOptions<
    object,
    Error,
    financeservicev1_CancelReceivableRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.receivableService.Cancel(req),
    ...options,
  });
}

// 拉取应收账龄报表。
export async function fetchAgingReportAr() {
  return queryClient.fetchQuery({
    queryKey: ['agingReportAr'],
    queryFn: () => apiClient.receivableService.AgingReport({}),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 枚举与工具函数
// ==============================

export const receivableStatusList = computed(() => [
  { value: 'PENDING', label: t('enum.finance.receivableStatus.Pending') },
  { value: 'PARTIAL', label: t('enum.finance.receivableStatus.Partial') },
  { value: 'SETTLED', label: t('enum.finance.receivableStatus.Settled') },
  { value: 'CANCELLED', label: t('enum.finance.receivableStatus.Cancelled') },
]);

export function receivableStatusToName(
  status: financeservicev1_Receivable_Status,
) {
  const values = receivableStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function receivableStatusToColor(
  status: financeservicev1_Receivable_Status,
) {
  switch (status) {
    case 'PENDING': {
      return 'orange';
    }
    case 'PARTIAL': {
      return 'blue';
    }
    case 'SETTLED': {
      return 'green';
    }
    case 'CANCELLED': {
      return 'gray';
    }
    default: {
      return 'gray';
    }
  }
}

export { centsToYuan } from './procurement';
