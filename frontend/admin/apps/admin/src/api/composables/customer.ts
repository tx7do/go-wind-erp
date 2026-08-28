import type {
  salesservicev1_Customer,
  salesservicev1_DeleteCustomerRequest,
  salesservicev1_ListCustomerResponse,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

// ==============================
// 客户管理
// ==============================

export function useListCustomers(
  query: PaginationQuery,
  options?: UseQueryOptions<salesservicev1_ListCustomerResponse, Error>,
) {
  return useQuery({
    queryKey: ['listCustomers', query],
    queryFn: () => apiClient.customerService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListCustomers(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listCustomers', params],
    queryFn: () => apiClient.customerService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreateCustomer(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.customerService.Create({
        data: { ...values } as salesservicev1_Customer,
      }),
    ...options,
  });
}

export function useUpdateCustomer(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.customerService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteCustomer(
  options?: UseMutationOptions<
    object,
    Error,
    salesservicev1_DeleteCustomerRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.customerService.Delete(req),
    ...options,
  });
}
