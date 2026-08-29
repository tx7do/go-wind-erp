<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  apiClient,
  centsToYuanMonthly,
  limitLabel,
  type billingservicev1_Plan as Plan,
} from '#/api';
import { $t } from '#/locales';

defineOptions({ name: 'BillingPlanManagement' });

const gridOptions: VxeGridProps<Plan> = {
  toolbarConfig: { custom: true, refresh: true, zoom: true },
  height: 'auto',
  pagerConfig: { enabled: false },
  proxyConfig: {
    ajax: {
      query: async () => {
        const resp = await apiClient.planAdminService.List({
          noPaging: true,
        } as any);
        return { items: resp?.items ?? [], total: resp?.items?.length ?? 0 };
      },
    },
  },
  columns: [
    { title: $t('page.billing.code'), field: 'code' },
    { title: $t('page.billing.name'), field: 'name' },
    {
      title: $t('page.billing.price'),
      field: 'priceCents',
      formatter: ({ cellValue }) => centsToYuanMonthly(cellValue),
    },
    {
      title: $t('page.billing.maxUsers'),
      field: 'maxUsers',
      formatter: ({ cellValue }) => limitLabel(cellValue as number),
    },
    {
      title: $t('page.billing.maxOrders'),
      field: 'maxOrdersMonthly',
      formatter: ({ cellValue }) => limitLabel(cellValue as number),
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.billing.planManagement')" />
  </Page>
</template>
