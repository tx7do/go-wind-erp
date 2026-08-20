<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  apiClient,
  centsToYuan,
  fetchListPayables,
  PaginationQuery,
  payableStatusList,
  payableStatusToColor,
  payableStatusToName,
  type financeservicev1_Payable as Payable,
} from '#/api';
import { $t } from '#/locales';

import PayableDrawer from './payable-drawer.vue';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'payableNumber',
      label: $t('page.payable.payableNumber'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'supplierCode',
      label: $t('page.payable.supplierCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.payable.status'),
      componentProps: {
        options: payableStatusList,
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Payable> = {
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
        return await fetchListPayables(
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
    { title: $t('page.payable.payableNumber'), field: 'payableNumber' },
    { title: $t('page.payable.poRef'), field: 'poRef' },
    { title: $t('page.payable.supplierCode'), field: 'supplierCode' },
    {
      title: $t('page.payable.amount'),
      field: 'amount',
      formatter: ({ cellValue }) => centsToYuan(cellValue),
    },
    {
      title: $t('page.payable.paidAmount'),
      field: 'paidAmount',
      formatter: ({ cellValue }) => centsToYuan(cellValue),
    },
    {
      title: $t('page.payable.status'),
      field: 'status',
      slots: { default: 'status' },
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 120,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: PayableDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

function openModal(create: boolean, row?: any) {
  drawerApi.setData({ create, row });
  drawerApi.open();
}

function handleCreate() {
  openModal(true);
}

// 详情抽屉承载记账/付款/取消动作。
function handleDetail(row: any) {
  openModal(false, row);
}

async function handleDelete(row: any) {
  try {
    await apiClient.payableService.Delete({ id: row.id });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.finance.payable')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.payable.button.create') }}
        </a-button>
      </template>

      <template #status="{ row }">
        <a-tag :color="payableStatusToColor(row.status)">
          {{ payableStatusToName(row.status) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideFilePenLine)"
          @click.stop="handleDetail(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.payable.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button
            danger
            type="link"
            :icon="h(LucideTrash2)"
            :disabled="row.status !== 'PENDING' || row.paidAmount > 0"
          />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
