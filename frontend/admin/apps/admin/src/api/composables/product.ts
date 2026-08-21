import type {
  productservicev1_DeleteProductRequest,
  productservicev1_GetProductRequest,
  productservicev1_ListProductResponse,
  productservicev1_Product,
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
// 仓库管理
// ==============================

export function useListProducts(
  query: PaginationQuery,
  options?: UseQueryOptions<productservicev1_ListProductResponse, Error>,
) {
  return useQuery({
    queryKey: ['listProducts', query],
    queryFn: () => apiClient.productService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListProducts(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listProducts', params],
    queryFn: () => apiClient.productService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetProduct(
  req: productservicev1_GetProductRequest,
  options?: UseQueryOptions<productservicev1_Product, Error>,
) {
  return useQuery({
    queryKey: ['getProduct', req],
    queryFn: () => apiClient.productService.Get(req),
    ...options,
  });
}

export function useCreateProduct(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.productService.Create({
        data: { ...values } as productservicev1_Product,
      }),
    ...options,
  });
}

export function useUpdateProduct(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.productService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteProduct(
  options?: UseMutationOptions<
    object,
    Error,
    productservicev1_DeleteProductRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.productService.Delete(req),
    ...options,
  });
}
