<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import {
  Page,
  useVbenDrawer,
  type VbenFormProps,
} from '@vben/common-ui';
import {
  LucideArrowLeftRight,
  LucideTrash2,
} from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  apiClient,
  derivedStateList,
  derivedStateToColor,
  derivedStateToName,
  fetchListStockPickings,
  PaginationQuery,
  pickingTypeList,
  pickingTypeToColor,
  pickingTypeToName,
  type inventoryservicev1_StockPicking as StockPicking,
} from '#/api';
import { $t } from '#/locales';

import StockPickingDrawer from './stock_movement-drawer.vue';
import AdjustmentDrawerComponent from './adjustment-drawer.vue';
import TransferDrawerComponent from './transfer-drawer.vue';

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'pickingNumber',
      label: $t('page.stockPicking.pickingNumber'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'pickingType',
      label: $t('page.stockPicking.pickingType'),
      componentProps: {
        options: pickingTypeList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'derivedState',
      label: $t('page.stockPicking.derivedState'),
      componentProps: {
        options: derivedStateList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<StockPicking> = {
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
        return await fetchListStockPickings(
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
    { title: $t('page.stockPicking.pickingNumber'), field: 'pickingNumber' },
    {
      title: $t('page.stockPicking.pickingType'),
      field: 'pickingType',
      slots: { default: 'pickingType' },
    },
    {
      title: $t('page.stockPicking.derivedState'),
      field: 'derivedState',
      slots: { default: 'derivedState' },
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
  connectedComponent: StockPickingDrawer,

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

const [AdjustmentDrawer, adjustmentDrawerApi] = useVbenDrawer({
  connectedComponent: AdjustmentDrawerComponent,

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

function handleAdjustment() {
  adjustmentDrawerApi.open();
}

async function handleDelete(row: any) {
  const state = row?.derivedState as string | undefined;
  if (state !== 'DRAFT' && state !== 'CANCELLED') {
    notification.error({
      message: $t('page.stockPicking.deleteNotAllowed'),
    });
    return;
  }
  try {
    await apiClient.stockPickingService.Delete({ id: row.id });

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

// 拣货执行动作：DRAFT→Confirm（锁定计划），CONFIRMED→Validate（执行回写库存）。
// 服务端是权威守卫，前端按钮按当前派生态显隐。
async function handleConfirmPicking(row: any) {
  try {
    await apiClient.stockPickingService.Confirm({ id: row.id });

    notification.success({
      message: $t('ui.notification.operation_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}

async function handleValidatePicking(row: any) {
  try {
    await apiClient.stockPickingService.Validate({
      id: row.id,
      lotAssignments: undefined,
    });

    notification.success({
      message: $t('ui.notification.operation_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.wms.stockPicking')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.stockPicking.button.create') }}
        </a-button>
        <a-button class="mr-2" :icon="h(LucideArrowLeftRight)" @click="handleTransfer">
          {{ $t('page.stockPicking.button.transfer') }}
        </a-button>
        <a-button class="mr-2" @click="handleAdjustment">
          {{ $t('page.stockPicking.button.adjustment') }}
        </a-button>
      </template>

      <template #pickingType="{ row }">
        <a-tag :color="pickingTypeToColor(row.pickingType)">
          {{ pickingTypeToName(row.pickingType) }}
        </a-tag>
      </template>
      <template #derivedState="{ row }">
        <a-tag :color="derivedStateToColor(row.derivedState)">
          {{ derivedStateToName(row.derivedState) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          v-if="row.derivedState === 'DRAFT'"
          class="mr-1"
          size="small"
          type="link"
          @click="handleConfirmPicking(row)"
        >
          {{ $t('page.stockPicking.button.confirm') }}
        </a-button>
        <a-button
          v-else-if="row.derivedState === 'CONFIRMED'"
          class="mr-1"
          size="small"
          type="link"
          @click="handleValidatePicking(row)"
        >
          {{ $t('page.stockPicking.button.validate') }}
        </a-button>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.stockPicking.moduleName'),
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
    <AdjustmentDrawer />
  </Page>
</template>
