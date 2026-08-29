<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import type { EchartsUIType } from '@vben/plugins/echarts';

import { ref, shallowRef, watch } from 'vue';

import { Page, type VbenFormProps } from '@vben/common-ui';

import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { apiClient, centsToYuan } from '#/api';
import { $t } from '#/locales';
import { exportRowsToExcel } from '#/utils/export-excel';

defineOptions({ name: 'SalesRankingReport' });

const dimensionOptions = [
  { value: 'SKU', label: $t('page.salesRanking.bySku') },
  { value: 'CUSTOMER', label: $t('page.salesRanking.byCustomer') },
];

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Select',
      fieldName: 'dimension',
      label: $t('page.salesRanking.bySku'),
      defaultValue: 'SKU',
      componentProps: {
        options: dimensionOptions,
      },
    },
    {
      component: 'DatePicker',
      fieldName: 'fromDate',
      label: $t('page.trialBalance.fromDate'),
      componentProps: {
        valueFormat: 'YYYY-MM-DD',
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
      },
    },
    {
      component: 'DatePicker',
      fieldName: 'toDate',
      label: $t('page.trialBalance.toDate'),
      componentProps: {
        valueFormat: 'YYYY-MM-DD',
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'limit',
      label: 'Top N',
      defaultValue: 10,
      componentProps: {
        options: [
          { value: 5, label: 'Top 5' },
          { value: 10, label: 'Top 10' },
          { value: 20, label: 'Top 20' },
        ],
      },
    },
  ],
};

// 图表数据源：query 回调存一份供 echarts 渲染（与表格同源）。
const chartItems = ref<any[]>([]);
// 行数状态（导出按钮 disabled 用，避免模板里调 grid 实例方法）。
const rowCount = ref(0);
const currentDimension = ref('SKU');

const gridOptions: VxeGridProps<any> = {
  toolbarConfig: { custom: true, refresh: true, zoom: true },
  height: 'auto',
  pagerConfig: { enabled: false },
  proxyConfig: {
    ajax: {
      query: async (_params, formValues) => {
        const dimension = (formValues?.dimension as string) ?? 'SKU';
        currentDimension.value = dimension;
        const resp = await apiClient.financeReportService.GetSalesRanking({
          dimension,
          fromDate: formValues?.fromDate
            ? new Date(`${formValues.fromDate}T00:00:00Z`)
            : undefined,
          toDate: formValues?.toDate
            ? new Date(`${formValues.toDate}T23:59:59Z`)
            : undefined,
          limit: Number(formValues?.limit ?? 10),
        } as any);
        const items = resp?.items ?? [];
        chartItems.value = items;
        rowCount.value = items.length;
        return { items, total: items.length };
      },
    },
  },
  columns: [
    { title: $t('page.salesRanking.rank'), type: 'seq', width: 70 },
    { title: $t('page.salesRanking.name'), field: 'label' },
    {
      title: $t('page.salesRanking.quantity'),
      field: 'quantity',
      width: 110,
      align: 'right',
    },
    {
      title: $t('page.salesRanking.amount'),
      field: 'amount',
      width: 130,
      align: 'right',
      formatter: ({ cellValue }) => centsToYuan(Number(cellValue ?? 0)),
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const chartRef = shallowRef<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

watch(chartItems, (val) => {
  if (!val.length) return;
  renderEcharts({
    grid: { top: 20, right: 30, bottom: 30, left: 130 },
    xAxis: { type: 'value' },
    yAxis: {
      type: 'category',
      inverse: true,
      data: val.map((i) => i.label ?? i.key),
    },
    series: [
      {
        type: 'bar',
        data: val.map((i) => Number(i.amount ?? 0) / 100),
        label: { show: true, position: 'right' },
      },
    ],
  });
});

function handleExport() {
  const tableData = gridApi.grid?.getTableData().fullData ?? [];
  exportRowsToExcel(
    `销售排行_${currentDimension.value}_${new Date().toISOString().slice(0, 10)}`,
    '销售排行',
    [
      $t('page.salesRanking.rank'),
      $t('page.salesRanking.name'),
      $t('page.salesRanking.quantity'),
      $t('page.salesRanking.amount'),
    ],
    tableData.map((it: any, idx: number) => [
      idx + 1,
      it.label ?? it.key ?? '',
      Number(it.quantity ?? 0),
      centsToYuan(Number(it.amount ?? 0)),
    ]),
  );
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.finance.salesRanking')">
      <template #toolbar-tools>
        <a-button
          class="mr-2"
          :disabled="rowCount === 0"
          @click="handleExport"
        >
          {{ $t('page.salesRanking.export') }}
        </a-button>
      </template>
    </Grid>
    <a-card class="mt-3" :title="$t('page.salesRanking.chartTitle')">
      <EchartsUI ref="chartRef" height="320px" />
    </a-card>
  </Page>
</template>
