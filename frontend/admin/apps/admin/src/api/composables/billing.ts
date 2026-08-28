import type {
  billingservicev1_ChangePlanRequest,
  billingservicev1_SubscriptionUsage,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { PaginationQuery } from '#/transport/rest';

// ==============================
// 订阅与套餐（透明定价）
// ==============================

export async function fetchMySubscription() {
  return queryClient.fetchQuery({
    queryKey: ['mySubscription'],
    queryFn: () => apiClient.billingService.GetMySubscription({}),
    staleTime: 0,
    retry: 0,
  });
}

export function useMySubscription(
  options?: UseQueryOptions<billingservicev1_SubscriptionUsage, Error>,
) {
  return useQuery({
    queryKey: ['mySubscription'],
    queryFn: () => apiClient.billingService.GetMySubscription({}),
    ...options,
  });
}

export async function fetchListPlans(params?: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPlans', params],
    queryFn: () =>
      apiClient.billingService.ListPlans(
        (params ?? new PaginationQuery({})).toRawParams(),
      ),
    staleTime: 0,
    retry: 0,
  });
}

export function useChangePlan(
  options?: UseMutationOptions<object, Error, { planCode: string }>,
) {
  return useMutation({
    mutationFn: ({ planCode }: { planCode: string }) =>
      apiClient.billingService.ChangePlan({
        planCode,
      } as billingservicev1_ChangePlanRequest),
    ...options,
  });
}

export function useRenew(options?: UseMutationOptions<object, Error, void>) {
  return useMutation({
    mutationFn: () => apiClient.billingService.Renew({}),
    ...options,
  });
}

export function centsToYuanMonthly(cents?: number | null): string {
  if (!cents || cents <= 0) {
    return '¥0/月';
  }
  return `¥${(cents / 100).toFixed(0)}/月`;
}

export function limitLabel(v?: number | null): string {
  if (!v || v <= 0) {
    return '∞';
  }
  return `${v}`;
}
