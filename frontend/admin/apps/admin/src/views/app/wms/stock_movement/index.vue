<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import {
  Page,
  useVbenDrawer,
  useVbenModal,
  type VbenFormProps,
} from '@vben/common-ui';
import {
  LucideArrowLeftRight,
  LucideRotateCcw,
  LucideTrash2,
} from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  apiClient,
  fetchListStockMovements,
  movementTypeList,
  movementTypeToColor,
  movementTypeToName,
  PaginationQuery,
  type inventoryservicev1_StockMovement as StockMovement,
} from '#/api';
import { $t } from '#/locales';

import ReverseModalComponent from './reverse-modal.vue';
import StockMovementDrawer from './stock_movement-drawer.vue';
import TransferDrawerComponent from './transfer-drawer.vue';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'warehouseCode',
      label: $t('page.stockMovement.warehouseCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'skuCode',
      label: $t('page.stockMovement.skuCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'movementType',
      label: $t('page.stockMovement.movementType'),
      componentProps: {
        options: movementTypeList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<StockMovement> = {
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
        return await fetchListStockMovements(
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
    { title: $t('page.stockMovement.warehouseCode'), field: 'warehouseCode' },
    { title: $t('page.stockMovement.skuCode'), field: 'skuCode' },
    { title: $t('page.stockMovement.delta'), field: 'delta' },
    {
      title: $t('page.stockMovement.movementType'),
      field: 'movementType',
      slots: { default: 'movementType' },
    },
    { title: $t('page.stockMovement.quantityBefore'), field: 'quantityBefore' },
    { title: $t('page.stockMovement.quantityAfter'), field: 'quantityAfter' },
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
  connectedComponent: StockMovementDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

const [TransferDrawer, transferDrawerApi] = useVbenDrawer({
  connectedComponent: TransferDrawerComponent,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

const [ReverseModal, reverseModalApi] = useVbenModal({
  connectedComponent: ReverseModalComponent,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

function openModal(create: boolean, row?: any) {
  drawerApi.setData({
    create,
    row,
  });

  drawerApi.open();
}

function handleCreate() {
  openModal(true);
}

function handleTransfer() {
  transferDrawerApi.open();
}

function handleReverse(row: any) {
  reverseModalApi.setData({ id: row.id });
  reverseModalApi.open();
}

async function handleDelete(row: any) {
  try {
    await apiClient.stockMovementService.Delete({ id: row.id });

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
    <Grid :table-title="$t('menu.wms.stockMovement')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.stockMovement.button.create') }}
        </a-button>
        <a-button class="mr-2" :icon="h(LucideArrowLeftRight)" @click="handleTransfer">
          {{ $t('page.stockMovement.button.transfer') }}
        </a-button>
      </template>

      <template #movementType="{ row }">
        <a-tag :color="movementTypeToColor(row.movementType)">
          {{ movementTypeToName(row.movementType) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-tooltip :title="$t('page.stockMovement.button.reverse')">
          <a-button type="link" :icon="h(LucideRotateCcw)" @click="handleReverse(row)" />
        </a-tooltip>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.stockMovement.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
    <TransferDrawer />
    <ReverseModal />
  </Page>
</template>
