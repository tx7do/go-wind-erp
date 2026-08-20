<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  centsToYuan,
  fetchListPayments,
  PaginationQuery,
  paymentMethodToName,
  type financeservicev1_Payment as Payment,
} from '#/api';
import { $t } from '#/locales';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'payableId',
      label: $t('page.payment.payableId'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Payment> = {
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
        return await fetchListPayments(
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
    { title: $t('page.payment.paymentNumber'), field: 'paymentNumber' },
    { title: $t('page.payment.payableId'), field: 'payableId' },
    {
      title: $t('page.payment.amount'),
      field: 'amount',
      formatter: ({ cellValue }) => centsToYuan(cellValue),
    },
    {
      title: $t('page.payment.method'),
      field: 'method',
      formatter: ({ cellValue }) => paymentMethodToName(cellValue),
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
    <Grid :table-title="$t('menu.finance.payment')" />
  </Page>
</template>
