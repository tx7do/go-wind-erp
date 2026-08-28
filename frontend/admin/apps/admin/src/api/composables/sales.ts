import type {
  salesservicev1_ApproveSalesOrderRequest,
  salesservicev1_CancelSalesOrderRequest,
  salesservicev1_CompleteSalesOrderRequest,
  salesservicev1_DeleteSalesOrderRequest,
  salesservicev1_GetSalesOrderRequest,
  salesservicev1_ListSalesOrderResponse,
  salesservicev1_RejectSalesOrderRequest,
  salesservicev1_SalesOrder,
  salesservicev1_SalesOrderItem,
  salesservicev1_SalesOrder_Status,
  salesservicev1_SubmitSalesOrderRequest,
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
// 销售单管理
// ==============================

export function useListSalesOrders(
  query: PaginationQuery,
  options?: UseQueryOptions<salesservicev1_ListSalesOrderResponse, Error>,
) {
  return useQuery({
    queryKey: ['listSalesOrders', query],
    queryFn: () => apiClient.salesOrderService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListSalesOrders(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listSalesOrders', params],
    queryFn: () => apiClient.salesOrderService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetSalesOrder(
  req: salesservicev1_GetSalesOrderRequest,
  options?: UseQueryOptions<salesservicev1_SalesOrder, Error>,
) {
  return useQuery({
    queryKey: ['getSalesOrder', req],
    queryFn: () => apiClient.salesOrderService.Get(req),
    ...options,
  });
}

export function useCreateSalesOrder(
  options?: UseMutationOptions<
    object,
    Error,
    { customerCode: string; warehouseCode: string; remark?: string; items: Record<string, any>[] }
  >,
) {
  return useMutation({
    mutationFn: ({ customerCode, warehouseCode, remark, items }) =>
      apiClient.salesOrderService.Create({
        data: {
          customerCode,
          warehouseCode,
          remark,
          items: items as salesservicev1_SalesOrderItem[],
        } as salesservicev1_SalesOrder,
      }),
    ...options,
  });
}

export function useUpdateSalesOrder(
  options?: UseMutationOptions<
    object,
    Error,
    {
      id: number;
      customerCode: string;
      warehouseCode: string;
      remark?: string;
      items: Record<string, any>[];
    }
  >,
) {
  return useMutation({
    mutationFn: ({ id, customerCode, warehouseCode, remark, items }) =>
      apiClient.salesOrderService.Update({
        id,
        data: {
          customerCode,
          warehouseCode,
          remark,
          items: items as salesservicev1_SalesOrderItem[],
        },
        updateMask: 'customerCode,warehouseCode,remark,items',
      }),
    ...options,
  });
}

export function useDeleteSalesOrder(
  options?: UseMutationOptions<
    object,
    Error,
    salesservicev1_DeleteSalesOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.salesOrderService.Delete(req),
    ...options,
  });
}

export function useSubmitSalesOrder(
  options?: UseMutationOptions<
    object,
    Error,
    salesservicev1_SubmitSalesOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.salesOrderService.Submit(req),
    ...options,
  });
}

export function useApproveSalesOrder(
  options?: UseMutationOptions<
    object,
    Error,
    salesservicev1_ApproveSalesOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.salesOrderService.Approve(req),
    ...options,
  });
}

export function useRejectSalesOrder(
  options?: UseMutationOptions<
    object,
    Error,
    salesservicev1_RejectSalesOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.salesOrderService.Reject(req),
    ...options,
  });
}

export function useCancelSalesOrder(
  options?: UseMutationOptions<
    object,
    Error,
    salesservicev1_CancelSalesOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.salesOrderService.Cancel(req),
    ...options,
  });
}

export function useCompleteSalesOrder(
  options?: UseMutationOptions<
    object,
    Error,
    salesservicev1_CompleteSalesOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.salesOrderService.Complete(req),
    ...options,
  });
}

// ==============================
// 销售单状态枚举与工具函数
// ==============================

export const salesOrderStatusList = computed(() => [
  { value: 'DRAFT', label: t('enum.sales.status.Draft') },
  { value: 'SUBMITTED', label: t('enum.sales.status.Submitted') },
  { value: 'APPROVED', label: t('enum.sales.status.Approved') },
  { value: 'REJECTED', label: t('enum.sales.status.Rejected') },
  { value: 'COMPLETED', label: t('enum.sales.status.Completed') },
  { value: 'CANCELLED', label: t('enum.sales.status.Cancelled') },
]);

export function salesOrderStatusToName(
  status: salesservicev1_SalesOrder_Status,
) {
  const values = salesOrderStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function salesOrderStatusToColor(
  status: salesservicev1_SalesOrder_Status,
) {
  switch (status) {
    case 'DRAFT': {
      return 'default';
    }
    case 'SUBMITTED': {
      return 'orange';
    }
    case 'APPROVED': {
      return 'green';
    }
    case 'REJECTED': {
      return 'red';
    }
    case 'COMPLETED': {
      return 'blue';
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

