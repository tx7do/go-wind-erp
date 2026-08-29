import type {
  approvalservicev1_ApprovalFlow,
  approvalservicev1_ListApprovalFlowResponse,
} from '#/api/generated/admin/service/v1';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';
import { computed } from 'vue';

import { i18n } from '@vben/locales';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 审批流模板（多级审批：每租户每业务类型至多一条生效流程）
// ==============================

export function useListApprovalFlows(
  query: PaginationQuery,
  options?: UseQueryOptions<approvalservicev1_ListApprovalFlowResponse, Error>,
) {
  return useQuery({
    queryKey: ['listApprovalFlows', query],
    queryFn: () => apiClient.approvalFlowService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListApprovalFlows(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listApprovalFlows', params],
    queryFn: () => apiClient.approvalFlowService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export async function fetchApprovalFlow(id: number) {
  return queryClient.fetchQuery({
    queryKey: ['getApprovalFlow', id],
    queryFn: () => apiClient.approvalFlowService.Get({ id }),
    staleTime: 0,
    retry: 0,
  });
}

// 流程业务类型（与后端 validFlowBizTypes 对齐）
export const flowBizTypeList = computed(() => [
  { value: 'PURCHASE_ORDER', label: t('page.approvalFlow.bizType.PURCHASE_ORDER') },
  { value: 'SALES_ORDER', label: t('page.approvalFlow.bizType.SALES_ORDER') },
  { value: 'PAYMENT', label: t('page.approvalFlow.bizType.PAYMENT') },
  { value: 'RECEIPT', label: t('page.approvalFlow.bizType.RECEIPT') },
]);

export function flowBizTypeToName(bizType?: string) {
  const matched = flowBizTypeList.value.find((i) => i.value === bizType);
  return matched ? matched.label : (bizType ?? '');
}

export type { approvalservicev1_ApprovalFlow as ApprovalFlow };
