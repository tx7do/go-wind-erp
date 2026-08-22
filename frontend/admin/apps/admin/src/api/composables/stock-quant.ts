import type {
  inventoryservicev1_GetMovementTrendRequest,
  inventoryservicev1_GetStockQuantOverviewRequest,
  inventoryservicev1_GetStockQuantRequest,
  inventoryservicev1_ListStockQuantResponse,
  inventoryservicev1_MovementTrendResponse,
  inventoryservicev1_StockQuant,
  inventoryservicev1_StockQuantOverview,
} from '#/api/generated/admin/service/v1';

import {
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { type PaginationQuery } from '#/transport/rest';

// ==============================
// 库存量（只读：借鉴 Odoo stock.quant，quantity 仅由拣货校验变更）
// ==============================

export function useListStockQuants(
  query: PaginationQuery,
  options?: UseQueryOptions<inventoryservicev1_ListStockQuantResponse, Error>,
) {
  return useQuery({
    queryKey: ['listStockQuants', query],
    queryFn: () => apiClient.stockQuantService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListStockQuants(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listStockQuants', params],
    queryFn: () => apiClient.stockQuantService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetStockQuant(
  req: inventoryservicev1_GetStockQuantRequest,
  options?: UseQueryOptions<inventoryservicev1_StockQuant, Error>,
) {
  return useQuery({
    queryKey: ['getStockQuant', req],
    queryFn: () => apiClient.stockQuantService.Get(req),
    ...options,
  });
}

// ==============================
// 库存经营总览（看板聚合）
// ==============================

export function useGetStockQuantOverview(
  req: inventoryservicev1_GetStockQuantOverviewRequest,
  options?: UseQueryOptions<inventoryservicev1_StockQuantOverview, Error>,
) {
  return useQuery({
    queryKey: ['getStockQuantOverview', req],
    queryFn: () => apiClient.stockQuantService.GetOverview(req),
    ...options,
  });
}

export async function fetchStockQuantOverview(
  req: inventoryservicev1_GetStockQuantOverviewRequest,
) {
  return queryClient.fetchQuery({
    queryKey: ['getStockQuantOverview', req],
    queryFn: () => apiClient.stockQuantService.GetOverview(req),
    staleTime: 0,
    retry: 0,
  });
}

// 拉取近 30 日库存流水趋势（看板折线图用）。
export async function fetchMovementTrend(
  req: inventoryservicev1_GetMovementTrendRequest = {},
) {
  return queryClient.fetchQuery({
    queryKey: ['getMovementTrend'],
    queryFn: () => apiClient.stockQuantService.GetMovementTrend(req),
    staleTime: 0,
    retry: 0,
  });
}
