<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { centsToYuan, fetchAgingReportAr } from '#/api';
import { $t } from '#/locales';

defineOptions({ name: 'ReceivableAgingReport' });

// 桶顺序固定（与后端 label 对应），缺失桶补零以保持表格完整。
const orderedBuckets = [
  'overdue',
  '0_7',
  '8_30',
  '31_90',
  'over_90',
  'no_due_date',
] as const;

const gridOptions: VxeGridProps<any> = {
  toolbarConfig: { custom: true, refresh: true, zoom: true },
  height: 'auto',
  pagerConfig: { enabled: false },
  proxyConfig: {
    ajax: {
      query: async () => {
        const resp = await fetchAgingReportAr();
        const present = new Map<string, any>();
        for (const b of resp?.buckets ?? []) {
          if (b?.bucket) {
            present.set(b.bucket, b);
          }
        }
        // 缺失桶补零——固定顺序输出。
        const items = orderedBuckets.map(
          (label) =>
            present.get(label) ?? {
              bucket: label,
              count: 0,
              totalAmount: 0,
            },
        );
        return { items, total: items.length };
      },
    },
  },
  columns: [
    {
      title: $t('page.agingAr.bucket'),
      field: 'bucket',
      slots: { default: 'bucket' },
      width: 160,
    },
    { title: $t('page.agingAr.count'), field: 'count', width: 120 },
    {
      title: $t('page.agingAr.totalAmount'),
      field: 'totalAmount',
      formatter: ({ cellValue }) => centsToYuan(cellValue as number),
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.agingAr.title')">
      <template #bucket="{ row }">
        <a-tag :color="row.bucket === 'overdue' ? 'red' : 'default'">
          { $t('page.agingAr.bucketLabel.' + row.bucket) }
        </a-tag>
      </template>
      <template #toolbar-tools>
        <span class="px-2 text-xs text-gray-400">
          { $t('page.agingAr.disclaimer') }
        </span>
      </template>
    </Grid>
  </Page>
</template>
