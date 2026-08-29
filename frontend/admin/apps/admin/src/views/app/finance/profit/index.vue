<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { apiClient, centsToYuan } from '#/api';
import { $t } from '#/locales';

defineOptions({ name: 'ProfitReport' });

const gridOptions: VxeGridProps<any> = {
  toolbarConfig: { custom: true, refresh: true, zoom: true },
  height: 'auto',
  pagerConfig: { enabled: false },
  proxyConfig: {
    ajax: {
      query: async () => {
        const resp = await apiClient.financeReportService.ProfitReport({});
        const items = resp?.items ?? [];
        return { items, total: items.length };
      },
    },
  },
  columns: [
    { title: $t('page.profit.month'), field: 'month', width: 120 },
    {
      title: $t('page.profit.revenue'),
      field: 'revenue',
      formatter: ({ cellValue }) => centsToYuan(cellValue as number),
    },
    {
      title: $t('page.profit.cogs'),
      field: 'cogs',
      formatter: ({ cellValue }) => centsToYuan(cellValue as number),
    },
    {
      title: $t('page.profit.profit'),
      field: 'profit',
      formatter: ({ cellValue }) => centsToYuan(cellValue as number),
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.profit.title')">
      <template #toolbar-tools>
        <span class="px-2 text-xs text-gray-400">
          {{ $t('page.profit.disclaimer') }}
        </span>
      </template>
    </Grid>
  </Page>
</template>
