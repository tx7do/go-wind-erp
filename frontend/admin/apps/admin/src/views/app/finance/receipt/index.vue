<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  centsToYuan,
  fetchListReceipts,
  PaginationQuery,
  receiptMethodToName,
  receiptStatusToColor,
  receiptStatusToName,
  type financeservicev1_Receipt as Receipt,
} from '#/api';
import { $t } from '#/locales';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'receivableId',
      label: $t('page.receipt.receivableId'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Receipt> = {
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
      query: async ({ page }, formValues) => {
        return await fetchListReceipts(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.receipt.receiptNumber'), field: 'receiptNumber' },
    { title: $t('page.receipt.receivableId'), field: 'receivableId' },
    {
      title: $t('page.receipt.amount'),
      field: 'amount',
      formatter: ({ cellValue }) => centsToYuan(cellValue),
    },
    {
      title: $t('page.receipt.method'),
      field: 'method',
      formatter: ({ cellValue }) => receiptMethodToName(cellValue),
    },
    {
      title: $t('page.receipt.status'),
      field: 'status',
      slots: { default: 'status' },
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    { title: $t('ui.table.remark'), field: 'remark' },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.finance.receipt')">
      <template #status="{ row }">
        <a-tag :color="receiptStatusToColor(row.status)">
          {{ receiptStatusToName(row.status) }}
        </a-tag>
      </template>
    </Grid>
  </Page>
</template>
