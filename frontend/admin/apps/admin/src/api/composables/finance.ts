import type {
  financeservicev1_CancelPayableRequest,
  financeservicev1_DeletePayableRequest,
  financeservicev1_GetPayableRequest,
  financeservicev1_GetPaymentRequest,
  financeservicev1_ListPayableResponse,
  financeservicev1_ListPaymentResponse,
  financeservicev1_Payable,
  financeservicev1_Payable_Status,
  financeservicev1_Payment,
  financeservicev1_Payment_Method,
  financeservicev1_Payment_Status,
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
// 应付单管理
// ==============================

export function useListPayables(
  query: PaginationQuery,
  options?: UseQueryOptions<financeservicev1_ListPayableResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPayables', query],
    queryFn: () => apiClient.payableService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPayables(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPayables', params],
    queryFn: () => apiClient.payableService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPayable(
  req: financeservicev1_GetPayableRequest,
  options?: UseQueryOptions<financeservicev1_Payable, Error>,
) {
  return useQuery({
    queryKey: ['getPayable', req],
    queryFn: () => apiClient.payableService.Get(req),
    ...options,
  });
}

export function useCreatePayable(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.payableService.Create({
        data: { ...values } as financeservicev1_Payable,
      }),
    ...options,
  });
}

export function useDeletePayable(
  options?: UseMutationOptions<
    object,
    Error,
    financeservicev1_DeletePayableRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.payableService.Delete(req),
    ...options,
  });
}

export function useCancelPayable(
  options?: UseMutationOptions<
    object,
    Error,
    financeservicev1_CancelPayableRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.payableService.Cancel(req),
    ...options,
  });
}

// 拉取应付账龄报表。
export async function fetchAgingReport() {
  return queryClient.fetchQuery({
    queryKey: ['agingReport'],
    queryFn: () => apiClient.payableService.AgingReport({}),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 付款管理（append-only 台账）
// ==============================

export function useListPayments(
  query: PaginationQuery,
  options?: UseQueryOptions<financeservicev1_ListPaymentResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPayments', query],
    queryFn: () => apiClient.paymentService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPayments(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPayments', params],
    queryFn: () => apiClient.paymentService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPayment(
  req: financeservicev1_GetPaymentRequest,
  options?: UseQueryOptions<financeservicev1_Payment, Error>,
) {
  return useQuery({
    queryKey: ['getPayment', req],
    queryFn: () => apiClient.paymentService.Get(req),
    ...options,
  });
}

export function useCreatePayment(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.paymentService.Create({
        data: { ...values } as financeservicev1_Payment,
      }),
    ...options,
  });
}

// ==============================
// 枚举与工具函数
// ==============================

export const payableStatusList = computed(() => [
  { value: 'PENDING', label: t('enum.finance.payableStatus.Pending') },
  { value: 'PARTIAL', label: t('enum.finance.payableStatus.Partial') },
  { value: 'SETTLED', label: t('enum.finance.payableStatus.Settled') },
  { value: 'CANCELLED', label: t('enum.finance.payableStatus.Cancelled') },
]);

export function payableStatusToName(status: financeservicev1_Payable_Status) {
  const values = payableStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function payableStatusToColor(status: financeservicev1_Payable_Status) {
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

export const paymentStatusList = computed(() => [
  { value: 'PENDING', label: t('enum.finance.paymentStatus.Pending') },
  { value: 'APPLIED', label: t('enum.finance.paymentStatus.Applied') },
  { value: 'REJECTED', label: t('enum.finance.paymentStatus.Rejected') },
]);

export function paymentStatusToName(
  status?: financeservicev1_Payment_Status,
) {
  const values = paymentStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function paymentStatusToColor(
  status?: financeservicev1_Payment_Status,
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

export const paymentMethodList = computed(() => [
  { value: 'BANK_TRANSFER', label: t('enum.finance.paymentMethod.BankTransfer') },
  { value: 'CASH', label: t('enum.finance.paymentMethod.Cash') },
  { value: 'CHECK', label: t('enum.finance.paymentMethod.Check') },
  { value: 'OTHER', label: t('enum.finance.paymentMethod.Other') },
]);

export function paymentMethodToName(method: financeservicev1_Payment_Method) {
  const values = paymentMethodList.value;
  const matchedItem = values.find((item) => item.value === method);
  return matchedItem ? matchedItem.label : '';
}

export { centsToYuan } from './procurement';


// ==============================
// 经营汇总（驾驶舱：本月收入/成本/利润 + 应收应付余额）
// ==============================

export async function fetchFinanceSummary() {
  return queryClient.fetchQuery({
    queryKey: ['financeSummary'],
    queryFn: () => apiClient.financeReportService.GetFinanceSummary({}),
    staleTime: 0,
    retry: 0,
  });
}
