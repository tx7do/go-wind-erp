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
  fetchListPurchaseOrders,
  PaginationQuery,
  purchaseOrderStatusList,
  purchaseOrderStatusToColor,
  purchaseOrderStatusToName,
  type procurementservicev1_PurchaseOrder as PurchaseOrder,
} from '#/api';
import { $t } from '#/locales';

import PurchaseOrderDrawer from './purchase_order-drawer.vue';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'poNumber',
      label: $t('page.purchaseOrder.poNumber'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'supplierCode',
      label: $t('page.purchaseOrder.supplierCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.purchaseOrder.status'),
      componentProps: {
        options: purchaseOrderStatusList,
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<PurchaseOrder> = {
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
        return await fetchListPurchaseOrders(
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
    { title: $t('page.purchaseOrder.poNumber'), field: 'poNumber' },
    { title: $t('page.purchaseOrder.supplierCode'), field: 'supplierCode' },
    {
      title: $t('page.purchaseOrder.status'),
      field: 'status',
      slots: { default: 'status' },
    },
    {
      title: $t('page.purchaseOrder.totalAmount'),
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
  connectedComponent: PurchaseOrderDrawer,

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

async function handleDelete(row: any) {
  try {
    await apiClient.purchaseOrderService.Delete({ id: row.id });

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
    <Grid :table-title="$t('menu.procurement.purchaseOrder')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.purchaseOrder.button.create') }}
        </a-button>
      </template>

      <template #status="{ row }">
        <a-tag :color="purchaseOrderStatusToColor(row.status)">
          {{ purchaseOrderStatusToName(row.status) }}
        </a-tag>
      </template>
      <template #action="{ row }">
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
              moduleName: $t('page.purchaseOrder.moduleName'),
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
