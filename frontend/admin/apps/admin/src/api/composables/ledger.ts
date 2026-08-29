import type {
  financeservicev1_Account,
  financeservicev1_GetTrialBalanceRequest,
  financeservicev1_ListAccountResponse,
  financeservicev1_ListJournalEntryRequest,
  financeservicev1_ListJournalEntryResponse,
  financeservicev1_TrialBalanceResponse,
} from '#/api/generated/admin/service/v1';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';
import { computed } from 'vue';

import { i18n } from '@vben/locales';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

const t = i18n.global.t;

// ==============================
// 简易总账（科目/凭证/余额表，凭证由业务事件自动生成）
// ==============================

export function useListAccounts(
  options?: UseQueryOptions<financeservicev1_ListAccountResponse, Error>,
) {
  return useQuery({
    queryKey: ['listAccounts'],
    queryFn: () =>
      apiClient.accountingService.ListAccounts({ noPaging: true } as any),
    ...options,
  });
}

export async function fetchTrialBalance(
  req: financeservicev1_GetTrialBalanceRequest,
) {
  return queryClient.fetchQuery({
    queryKey: ['trialBalance', req.fromDate ?? '', req.toDate ?? ''],
    queryFn: () => apiClient.accountingService.GetTrialBalance(req),
    staleTime: 0,
    retry: 0,
  }) as Promise<financeservicev1_TrialBalanceResponse>;
}

export async function fetchJournalEntries(
  req: financeservicev1_ListJournalEntryRequest,
) {
  return queryClient.fetchQuery({
    queryKey: ['journalEntries', req],
    queryFn: () => apiClient.accountingService.ListJournalEntries(req),
    staleTime: 0,
    retry: 0,
  }) as Promise<financeservicev1_ListJournalEntryResponse>;
}

// 科目类别/方向本地化
export const accountCategoryList = computed(() => [
  { value: 'ASSET', label: t('enum.account.category.ASSET') },
  { value: 'LIABILITY', label: t('enum.account.category.LIABILITY') },
  { value: 'EQUITY', label: t('enum.account.category.EQUITY') },
  { value: 'REVENUE', label: t('enum.account.category.REVENUE') },
  { value: 'EXPENSE', label: t('enum.account.category.EXPENSE') },
]);

export function accountCategoryToName(category?: string) {
  const matched = accountCategoryList.value.find((i) => i.value === category);
  return matched ? matched.label : (category ?? '');
}

// 分转元显示（两位小数）。
export function centsToYuanLedger(cents: null | number | undefined): string {
  if (cents === null || cents === undefined) return '-';
  return (cents / 100).toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export type {
  financeservicev1_Account as LedgerAccount,
  financeservicev1_TrialBalanceResponse as TrialBalanceResponse,
};
