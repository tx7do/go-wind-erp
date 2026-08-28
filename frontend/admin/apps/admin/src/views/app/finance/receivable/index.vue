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
  fetchListReceivables,
  PaginationQuery,
  receivableStatusList,
  receivableStatusToColor,
  receivableStatusToName,
  type financeservicev1_Receivable as Receivable,
} from '#/api';
import { $t } from '#/locales';

import ReceivableDrawer from './receivable-drawer.vue';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'receivableNumber',
      label: $t('page.receivable.receivableNumber'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'customerCode',
      label: $t('page.receivable.customerCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.receivable.status'),
      componentProps: {
        options: receivableStatusList,
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Receivable> = {
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
        return await fetchListReceivables(
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
    { title: $t('page.receivable.receivableNumber'), field: 'receivableNumber' },
    { title: $t('page.receivable.soRef'), field: 'soRef' },
    { title: $t('page.receivable.customerCode'), field: 'customerCode' },
    {
      title: $t('page.receivable.amount'),
      field: 'amount',
      formatter: ({ cellValue }) => centsToYuan(cellValue),
    },
    {
      title: $t('page.receivable.paidAmount'),
      field: 'paidAmount',
      formatter: ({ cellValue }) => centsToYuan(cellValue),
    },
    {
      title: $t('page.receivable.status'),
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
  connectedComponent: ReceivableDrawer,

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

// 详情抽屉承载记账/收款/取消动作。
function handleDetail(row: any) {
  openModal(false, row);
}

async function handleDelete(row: any) {
  try {
    await apiClient.receivableService.Delete({ id: row.id });

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
    <Grid :table-title="$t('menu.finance.receivable')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.receivable.button.create') }}
        </a-button>
      </template>

      <template #status="{ row }">
        <a-tag :color="receivableStatusToColor(row.status)">
          {{ receivableStatusToName(row.status) }}
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
              moduleName: $t('page.receivable.moduleName'),
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
