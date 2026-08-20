import type {
  approvalservicev1_ApprovalRequest,
  approvalservicev1_ApprovalRequest_Status,
  approvalservicev1_DeleteApprovalRequestRequest,
  approvalservicev1_GetApprovalRequestRequest,
  approvalservicev1_ListApprovalRequestResponse,
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
// 审批请求管理
// ==============================

export function useListApprovalRequests(
  query: PaginationQuery,
  options?: UseQueryOptions<
    approvalservicev1_ListApprovalRequestResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listApprovalRequests', query],
    queryFn: () => apiClient.approvalRequestService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListApprovalRequests(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listApprovalRequests', params],
    queryFn: () => apiClient.approvalRequestService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetApprovalRequest(
  req: approvalservicev1_GetApprovalRequestRequest,
  options?: UseQueryOptions<approvalservicev1_ApprovalRequest, Error>,
) {
  return useQuery({
    queryKey: ['getApprovalRequest', req],
    queryFn: () => apiClient.approvalRequestService.Get(req),
    ...options,
  });
}

export function useCreateApprovalRequest(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.approvalRequestService.Create({
        data: { ...values } as approvalservicev1_ApprovalRequest,
      }),
    ...options,
  });
}

export function useDeleteApprovalRequest(
  options?: UseMutationOptions<
    object,
    Error,
    approvalservicev1_DeleteApprovalRequestRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.approvalRequestService.Delete(req),
    ...options,
  });
}

// ==============================
// 审批动作
// ==============================

export function useApproveApprovalRequest(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; comment?: string }
  >,
) {
  return useMutation({
    mutationFn: ({ id, comment }: { id: number; comment?: string }) =>
      apiClient.approvalRequestService.Approve({ id, comment }),
    ...options,
  });
}

export function useRejectApprovalRequest(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; comment?: string }
  >,
) {
  return useMutation({
    mutationFn: ({ id, comment }: { id: number; comment?: string }) =>
      apiClient.approvalRequestService.Reject({ id, comment }),
    ...options,
  });
}

export function useCancelApprovalRequest(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }: { id: number }) =>
      apiClient.approvalRequestService.Cancel({ id }),
    ...options,
  });
}

// ==============================
// 审批状态枚举与工具函数
// ==============================

export const approvalStatusList = computed(() => [
  { value: 'PENDING', label: t('enum.approval.status.Pending') },
  { value: 'APPROVED', label: t('enum.approval.status.Approved') },
  { value: 'REJECTED', label: t('enum.approval.status.Rejected') },
  { value: 'CANCELLED', label: t('enum.approval.status.Cancelled') },
]);

export function approvalStatusToName(
  status: approvalservicev1_ApprovalRequest_Status,
) {
  const values = approvalStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function approvalStatusToColor(
  status: approvalservicev1_ApprovalRequest_Status,
) {
  switch (status) {
    case 'PENDING': {
      return 'orange';
    }
    case 'APPROVED': {
      return 'green';
    }
    case 'REJECTED': {
      return 'red';
    }
    case 'CANCELLED': {
      return 'gray';
    }
    default: {
      return 'gray';
    }
  }
}
