<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { fetchTrialBalance } from '#/api';
import { $t } from '#/locales';

defineOptions({ name: 'TrialBalanceManagement' });

const gridOptions: VxeGridProps<any> = {
  toolbarConfig: {
    custom: true,
    export: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },
  proxyConfig: {
    ajax: {
      query: async () => {
        const resp = await fetchTrialBalance({} as any);
        return { items: resp?.items ?? [], total: resp?.items?.length ?? 0 };
      },
    },
  },
  columns: [
    { title: $t('page.trialBalance.accountCode'), field: 'accountCode' },
    { title: $t('page.trialBalance.accountName'), field: 'accountName' },
    {
      title: $t('page.trialBalance.balance'),
      field: 'balance',
      formatter: 'formatDateTime',
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.finance.trialBalance')" />
  </Page>
</template>
