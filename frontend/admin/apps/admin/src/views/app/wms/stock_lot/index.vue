<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListStockLots,
  formatLotExpiry,
  lotStatusList,
  lotStatusToColor,
  lotStatusToName,
  PaginationQuery,
  type inventoryservicev1_StockLot as StockLot,
} from '#/api';
import { $t } from '#/locales';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'skuCode',
      label: $t('page.stockLot.skuCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'lotStatus',
      label: $t('page.stockLot.lotStatus'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        options: lotStatusList,
      },
    },
  ],
};

const gridOptions: VxeGridProps<StockLot> = {
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
        return await fetchListStockLots(
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
    { title: $t('page.stockLot.name'), field: 'name' },
    { title: $t('page.stockLot.skuCode'), field: 'skuCode' },
    {
      title: $t('page.stockLot.expiryDate'),
      field: 'expiryDate',
      formatter: ({ cellValue }) => formatLotExpiry(cellValue),
      width: 120,
    },
    {
      title: $t('page.stockLot.remainingQuantity'),
      field: 'remainingQuantity',
    },
    {
      title: $t('page.stockLot.lotStatus'),
      field: 'lotStatus',
      slots: { default: 'lotStatus' },
      width: 100,
    },
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
    <Grid :table-title="$t('menu.wms.stockLot')">
      <template #lotStatus="{ row }">
        <a-tag :color="lotStatusToColor(row.lotStatus)">
          {{ lotStatusToName(row.lotStatus) }}
        </a-tag>
      </template>
    </Grid>
  </Page>
</template>
