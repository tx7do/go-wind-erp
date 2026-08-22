<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListStockQuants,
  PaginationQuery,
  type inventoryservicev1_StockQuant as StockQuant,
} from '#/api';
import { $t } from '#/locales';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'locationId',
      label: $t('page.stockQuant.locationId'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'productCode',
      label: $t('page.stockQuant.productCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<StockQuant> = {
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
        return await fetchListStockQuants(
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
    { title: $t('page.stockQuant.locationId'), field: 'locationId' },
    { title: $t('page.stockQuant.productCode'), field: 'productCode' },
    { title: $t('page.stockQuant.quantity'), field: 'quantity' },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.wms.stockQuant')" />
  </Page>
</template>
