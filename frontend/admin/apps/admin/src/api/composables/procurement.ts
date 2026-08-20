import type {
  procurementservicev1_ApprovePurchaseOrderRequest,
  procurementservicev1_CancelPurchaseOrderRequest,
  procurementservicev1_CompletePurchaseOrderRequest,
  procurementservicev1_DeletePurchaseOrderRequest,
  procurementservicev1_DeleteSupplierRequest,
  procurementservicev1_GetPurchaseOrderRequest,
  procurementservicev1_GetSupplierRequest,
  procurementservicev1_ListPurchaseOrderResponse,
  procurementservicev1_ListSupplierResponse,
  procurementservicev1_PurchaseOrder,
  procurementservicev1_PurchaseOrder_Status,
  procurementservicev1_PurchaseOrderItem,
  procurementservicev1_RejectPurchaseOrderRequest,
  procurementservicev1_SubmitPurchaseOrderRequest,
  procurementservicev1_Supplier,
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
// 供应商管理
// ==============================

export function useListSuppliers(
  query: PaginationQuery,
  options?: UseQueryOptions<procurementservicev1_ListSupplierResponse, Error>,
) {
  return useQuery({
    queryKey: ['listSuppliers', query],
    queryFn: () => apiClient.supplierService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListSuppliers(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listSuppliers', params],
    queryFn: () => apiClient.supplierService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetSupplier(
  req: procurementservicev1_GetSupplierRequest,
  options?: UseQueryOptions<procurementservicev1_Supplier, Error>,
) {
  return useQuery({
    queryKey: ['getSupplier', req],
    queryFn: () => apiClient.supplierService.Get(req),
    ...options,
  });
}

export function useCreateSupplier(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.supplierService.Create({
        data: { ...values } as procurementservicev1_Supplier,
      }),
    ...options,
  });
}

export function useUpdateSupplier(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.supplierService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteSupplier(
  options?: UseMutationOptions<
    object,
    Error,
    procurementservicev1_DeleteSupplierRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.supplierService.Delete(req),
    ...options,
  });
}

// ==============================
// 采购单管理
// ==============================

export function useListPurchaseOrders(
  query: PaginationQuery,
  options?: UseQueryOptions<
    procurementservicev1_ListPurchaseOrderResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listPurchaseOrders', query],
    queryFn: () => apiClient.purchaseOrderService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPurchaseOrders(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPurchaseOrders', params],
    queryFn: () => apiClient.purchaseOrderService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPurchaseOrder(
  req: procurementservicev1_GetPurchaseOrderRequest,
  options?: UseQueryOptions<procurementservicev1_PurchaseOrder, Error>,
) {
  return useQuery({
    queryKey: ['getPurchaseOrder', req],
    queryFn: () => apiClient.purchaseOrderService.Get(req),
    ...options,
  });
}

export function useCreatePurchaseOrder(
  options?: UseMutationOptions<
    object,
    Error,
    { supplierCode: string; remark?: string; items: Record<string, any>[] }
  >,
) {
  return useMutation({
    mutationFn: ({ supplierCode, remark, items }) =>
      apiClient.purchaseOrderService.Create({
        data: {
          supplierCode,
          remark,
          items: items as procurementservicev1_PurchaseOrderItem[],
        } as procurementservicev1_PurchaseOrder,
      }),
    ...options,
  });
}

export function useUpdatePurchaseOrder(
  options?: UseMutationOptions<
    object,
    Error,
    {
      id: number;
      supplierCode: string;
      remark?: string;
      items: Record<string, any>[];
    }
  >,
) {
  return useMutation({
    mutationFn: ({ id, supplierCode, remark, items }) =>
      apiClient.purchaseOrderService.Update({
        id,
        data: {
          supplierCode,
          remark,
          items: items as procurementservicev1_PurchaseOrderItem[],
        },
        updateMask: 'supplierCode,remark,items',
      }),
    ...options,
  });
}

export function useDeletePurchaseOrder(
  options?: UseMutationOptions<
    object,
    Error,
    procurementservicev1_DeletePurchaseOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseOrderService.Delete(req),
    ...options,
  });
}

// ==============================
// 采购单动作
// ==============================

export function useSubmitPurchaseOrder(
  options?: UseMutationOptions<
    object,
    Error,
    procurementservicev1_SubmitPurchaseOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseOrderService.Submit(req),
    ...options,
  });
}

export function useApprovePurchaseOrder(
  options?: UseMutationOptions<
    object,
    Error,
    procurementservicev1_ApprovePurchaseOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseOrderService.Approve(req),
    ...options,
  });
}

export function useRejectPurchaseOrder(
  options?: UseMutationOptions<
    object,
    Error,
    procurementservicev1_RejectPurchaseOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseOrderService.Reject(req),
    ...options,
  });
}

export function useCancelPurchaseOrder(
  options?: UseMutationOptions<
    object,
    Error,
    procurementservicev1_CancelPurchaseOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseOrderService.Cancel(req),
    ...options,
  });
}

export function useCompletePurchaseOrder(
  options?: UseMutationOptions<
    object,
    Error,
    procurementservicev1_CompletePurchaseOrderRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseOrderService.Complete(req),
    ...options,
  });
}

// ==============================
// 采购单状态枚举与工具函数
// ==============================

export const purchaseOrderStatusList = computed(() => [
  { value: 'DRAFT', label: t('enum.procurement.status.Draft') },
  { value: 'SUBMITTED', label: t('enum.procurement.status.Submitted') },
  { value: 'APPROVED', label: t('enum.procurement.status.Approved') },
  { value: 'REJECTED', label: t('enum.procurement.status.Rejected') },
  { value: 'COMPLETED', label: t('enum.procurement.status.Completed') },
  { value: 'CANCELLED', label: t('enum.procurement.status.Cancelled') },
]);

export function purchaseOrderStatusToName(
  status: procurementservicev1_PurchaseOrder_Status,
) {
  const values = purchaseOrderStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function purchaseOrderStatusToColor(
  status: procurementservicev1_PurchaseOrder_Status,
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

// 分转元显示（两位小数）。
export function centsToYuan(cents: null | number | undefined): string {
  if (cents === null || cents === undefined) return '-';
  return (cents / 100).toFixed(2);
}
