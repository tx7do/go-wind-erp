import type {
  inventoryservicev1_ListStockLotResponse,
  inventoryservicev1_LotStatus,
} from '#/api/generated/admin/service/v1';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';
import { computed } from 'vue';

import { i18n } from '@vben/locales';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

const t = i18n.global.t;
import { type PaginationQuery } from '#/transport/rest';

// ==============================
// 批次库存（记录式批次/效期：余量由 move lines 聚合，仅收货时登记批次）
// ==============================

export function useListStockLots(
  query: PaginationQuery,
  options?: UseQueryOptions<inventoryservicev1_ListStockLotResponse, Error>,
) {
  return useQuery({
    queryKey: ['listStockLots', query],
    queryFn: () => apiClient.stockLotService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListStockLots(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listStockLots', params],
    queryFn: () => apiClient.stockLotService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 效期状态枚举与工具函数
// ==============================

export const lotStatusList = computed(() => [
  { value: 'LOT_NORMAL', label: t('enum.stockLot.Normal') },
  { value: 'LOT_EXPIRING', label: t('enum.stockLot.Expiring') },
  { value: 'LOT_EXPIRED', label: t('enum.stockLot.Expired') },
]);

export function lotStatusToName(status?: inventoryservicev1_LotStatus) {
  const matched = lotStatusList.value.find((item) => item.value === status);
  return matched ? matched.label : '';
}

export function lotStatusToColor(status?: inventoryservicev1_LotStatus) {
  switch (status) {
    case 'LOT_NORMAL': {
      return 'green';
    }
    case 'LOT_EXPIRING': {
      return 'orange';
    }
    case 'LOT_EXPIRED': {
      return 'red';
    }
    default: {
      return 'gray';
    }
  }
}

// 效期格式化（yyyy-MM-dd）；无期显示 —。
export function formatLotExpiry(expiry?: null | string): string {
  if (!expiry) return '—';
  const d = new Date(expiry);
  if (Number.isNaN(d.getTime())) return expiry;
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

