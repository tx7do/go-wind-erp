<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucidePrinter, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  apiClient,
  centsToYuan,
  fetchListSalesOrders,
  PaginationQuery,
  salesOrderStatusList,
  salesOrderStatusToColor,
  salesOrderStatusToName,
  type salesservicev1_SalesOrder as SalesOrder,
} from '#/api';
import { $t } from '#/locales';
import { printSalesOrderById } from '#/utils/order-print';

import SalesOrderDrawer from './sales_order-drawer.vue';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'soNumber',
      label: $t('page.salesOrder.soNumber'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'customerCode',
      label: $t('page.salesOrder.customerCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.salesOrder.status'),
      componentProps: {
        options: salesOrderStatusList,
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<SalesOrder> = {
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
        return await fetchListSalesOrders(
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
    { title: $t('page.salesOrder.soNumber'), field: 'soNumber' },
    { title: $t('page.salesOrder.customerCode'), field: 'customerCode' },
    {
      title: $t('page.salesOrder.status'),
      field: 'status',
      slots: { default: 'status' },
    },
    {
      title: $t('page.salesOrder.totalAmount'),
      field: 'totalAmount',
      formatter: ({ cellValue }) => centsToYuan(cellValue),
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
  connectedComponent: SalesOrderDrawer,

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

// 编辑仅对 DRAFT 有意义（服务端也仅放行 DRAFT），其余状态一律打开详情。
function handleEdit(row: any) {
  openModal(false, row);
}

// 行级打印：先拉完整单据（列表行不含明细）再打印。
async function handlePrint(row: any) {
  try {
    await printSalesOrderById(row.id);
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}

async function handleDelete(row: any) {
  try {
    await apiClient.salesOrderService.Delete({ id: row.id });

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
    <Grid :table-title="$t('menu.sales.salesOrder')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.salesOrder.button.create') }}
        </a-button>
      </template>

      <template #status="{ row }">
        <a-tag :color="salesOrderStatusToColor(row.status)">
          {{ salesOrderStatusToName(row.status) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucidePrinter)"
          :title="$t('page.salesOrder.button.print')"
          @click.stop="handlePrint(row)"
        />
        <a-button
          type="link"
          :icon="h(LucideFilePenLine)"
          @click.stop="handleEdit(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.salesOrder.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button
            danger
            type="link"
            :icon="h(LucideTrash2)"
            :disabled="
              row.status !== 'DRAFT' && row.status !== 'CANCELLED'
            "
          />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
