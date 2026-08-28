import type {
  financeservicev1_GetReceiptRequest,
  financeservicev1_ListReceiptResponse,
  financeservicev1_Receipt,
  financeservicev1_Receipt_Method,
  financeservicev1_Receipt_Status,
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
// 收款管理（append-only 台账）
// ==============================

export function useListReceipts(
  query: PaginationQuery,
  options?: UseQueryOptions<financeservicev1_ListReceiptResponse, Error>,
) {
  return useQuery({
    queryKey: ['listReceipts', query],
    queryFn: () => apiClient.receiptService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListReceipts(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listReceipts', params],
    queryFn: () => apiClient.receiptService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetReceipt(
  req: financeservicev1_GetReceiptRequest,
  options?: UseQueryOptions<financeservicev1_Receipt, Error>,
) {
  return useQuery({
    queryKey: ['getReceipt', req],
    queryFn: () => apiClient.receiptService.Get(req),
    ...options,
  });
}

export function useCreateReceipt(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.receiptService.Create({
        data: { ...values } as financeservicev1_Receipt,
      }),
    ...options,
  });
}

// ==============================
// 枚举与工具函数
// ==============================

export const receiptStatusList = computed(() => [
  { value: 'PENDING', label: t('enum.finance.receiptStatus.Pending') },
  { value: 'APPLIED', label: t('enum.finance.receiptStatus.Applied') },
  { value: 'REJECTED', label: t('enum.finance.receiptStatus.Rejected') },
]);

export function receiptStatusToName(
  status?: financeservicev1_Receipt_Status,
) {
  const values = receiptStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function receiptStatusToColor(
  status?: financeservicev1_Receipt_Status,
) {
  switch (status) {
    case 'PENDING': {
      return 'orange';
    }
    case 'APPLIED': {
      return 'green';
    }
    case 'REJECTED': {
      return 'red';
    }
    default: {
      return 'gray';
    }
  }
}

export const receiptMethodList = computed(() => [
  { value: 'BANK_TRANSFER', label: t('enum.finance.receiptMethod.BankTransfer') },
  { value: 'CASH', label: t('enum.finance.receiptMethod.Cash') },
  { value: 'CHECK', label: t('enum.finance.receiptMethod.Check') },
  { value: 'OTHER', label: t('enum.finance.receiptMethod.Other') },
]);

export function receiptMethodToName(
  method: financeservicev1_Receipt_Method,
) {
  const values = receiptMethodList.value;
  const matchedItem = values.find((item) => item.value === method);
  return matchedItem ? matchedItem.label : '';
}

export { centsToYuan } from './procurement';
