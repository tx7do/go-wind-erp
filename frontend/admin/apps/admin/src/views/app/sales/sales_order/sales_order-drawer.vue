<script lang="ts" setup>
import { computed, reactive, ref } from 'vue';

import { useVbenDrawer, useVbenModal } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import type { salesservicev1_SalesOrderItem } from '#/api/generated/admin/service/v1';

import { useVbenForm } from '#/adapter/form';
import {
  apiClient,
  centsToYuan,
  fetchListCustomers,
  fetchListWarehouses,
  salesOrderStatusToName,
} from '#/api';

import SalesReturnModalComponent from './sales-return-modal.vue';

const data = ref();

interface ItemRow {
  skuCode: string;
  quantity: null | number;
  unitPrice: null | number;
}

const items = reactive<ItemRow[]>([]);
const newItem = reactive<ItemRow>({
  skuCode: '',
  quantity: null,
  unitPrice: null,
});

const isCreate = computed(() => data.value?.create);
const isDraft = computed(() => data.value?.row?.status === 'DRAFT');
// APPROVED/COMPLETED 状态下可按明细退货（服务端 ApplyFulfillmentReturnTx
// 有状态门与防超退守卫，前端按钮按已履约数量显隐）。
const isReturnable = computed(
  () =>
    data.value?.row?.status === 'APPROVED' ||
    data.value?.row?.status === 'COMPLETED',
);

// 退货弹窗：收集退货数量，成功后关闭抽屉让列表刷新。
const [SalesReturnModal, salesReturnModalApi] = useVbenModal({
  connectedComponent: SalesReturnModalComponent,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      drawerApi.close();
    }
  },
});

function handleReturn(record: any) {
  salesReturnModalApi.setData({
    soId: data.value.row.id,
    soItemId: record.id,
    returnable: record.fulfilledQuantity ?? 0,
  });
  salesReturnModalApi.open();
}

const getTitle = computed(() =>
  isCreate.value
    ? $t('ui.modal.create', {
        moduleName: $t('page.salesOrder.moduleName'),
      })
    : `${data.value?.row?.soNumber ?? ''} — ${salesOrderStatusToName(
        data.value?.row?.status,
      )}`,
);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: {
    componentProps: {
      class: 'w-full',
    },
  },
  schema: [
    {
      component: 'ApiSelect',
      fieldName: 'customerCode',
      label: $t('page.salesOrder.customerCode'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await fetchListCustomers(
            new (await import('#/transport/rest')).PaginationQuery({
              paging: { page: 1, pageSize: 500 },
            }),
          );
          return result?.items ?? [];
        },
        afterFetch: (
          customers: { code?: string; enable?: boolean; name?: string }[],
        ) =>
          customers
            .filter((s) => s.enable)
            .map((s) => ({ label: `${s.code} — ${s.name}`, value: s.code })),
      },
    },
    {
      component: 'ApiSelect',
      fieldName: 'warehouseCode',
      label: $t('page.salesOrder.warehouseCode'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await fetchListWarehouses(
            new (await import('#/transport/rest')).PaginationQuery({
              paging: { page: 1, pageSize: 500 },
            }),
          );
          return result?.items ?? [];
        },
        afterFetch: (
          warehouses: { code?: string; enable?: boolean; name?: string }[],
        ) =>
          warehouses
            .filter((w) => w.enable)
            .map((w) => ({ label: `${w.code} — ${w.name}`, value: w.code })),
      },
    },
    {
      component: 'Textarea',
      fieldName: 'remark',
      label: $t('ui.table.remark'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
});

const [Drawer, drawerApi] = useVbenDrawer({
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {
    if (!isCreate.value && !isDraft.value) {
      drawerApi.close();
      return;
    }

    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    const values = await baseFormApi.getValues();
    const payloadItems = items
      .filter((i) => i.skuCode && i.quantity && i.quantity > 0)
      .map(
        (i): salesservicev1_SalesOrderItem => ({
          skuCode: i.skuCode,
          quantity: i.quantity as number,
          unitPrice: i.unitPrice ?? 0,
        }),
      );
    if (payloadItems.length === 0) {
      notification.warning({
        message: $t('page.salesOrder.itemsRequired'),
      });
      return;
    }

    setLoading(true);

    try {
      await (data.value?.create
        ? apiClient.salesOrderService.Create({
            data: {
              customerCode: values.customerCode,
              warehouseCode: values.warehouseCode,
              remark: values.remark,
              items: payloadItems,
            },
          })
        : apiClient.salesOrderService.Update({
            id: data.value.row.id,
            data: {
              customerCode: values.customerCode,
              warehouseCode: values.warehouseCode,
              remark: values.remark,
              items: payloadItems,
            },
            updateMask: 'customerCode,warehouseCode,remark,items',
          }));

      notification.success({
        message: data.value?.create
          ? $t('ui.notification.create_success')
          : $t('ui.notification.update_success'),
      });
    } catch {
      notification.error({
        message: data.value?.create
          ? $t('ui.notification.create_failed')
          : $t('ui.notification.update_failed'),
      });
    } finally {
      drawerApi.close();
      setLoading(false);
    }
  },

  async onOpenChange(isOpen: boolean) {
    if (isOpen) {
      data.value = drawerApi.getData<Record<string, any>>();

      // 列表行不含明细（明细仅在 Get 组装），打开时拉取完整单据
      // 供只读明细表/退货按钮使用。
      if (!data.value?.create && data.value?.row?.id) {
        try {
          const full = await apiClient.salesOrderService.Get({
            id: data.value.row.id,
          });
          if (full?.id) {
            data.value = { ...data.value, row: full };
          }
        } catch {
          // 拉取失败退回列表行数据
        }
      }

      items.splice(0, items.length);
      newItem.skuCode = '';
      newItem.quantity = null;
      newItem.unitPrice = null;

      baseFormApi.setValues({
        customerCode: data.value?.row?.customerCode,
        warehouseCode: data.value?.row?.warehouseCode,
        remark: data.value?.row?.remark,
      });

      setLoading(false);
    }
  },
});

async function handleAction(
  action: 'approve' | 'cancel' | 'complete' | 'reject' | 'submit',
) {
  setLoading(true);
  try {
    const id = data.value.row.id;
    if (action === 'submit') {
      await apiClient.salesOrderService.Submit({ id });
    } else if (action === 'approve') {
      await apiClient.salesOrderService.Approve({ id });
    } else if (action === 'reject') {
      await apiClient.salesOrderService.Reject({ id });
    } else if (action === 'cancel') {
      await apiClient.salesOrderService.Cancel({ id });
    } else {
      await apiClient.salesOrderService.Complete({ id });
    }

    notification.success({
      message: $t('ui.notification.operation_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  } finally {
    setLoading(false);
    drawerApi.close();
  }
}

function addItem() {
  if (!newItem.skuCode || !newItem.quantity || newItem.quantity <= 0) {
    notification.warning({ message: $t('page.salesOrder.itemInvalid') });
    return;
  }
  items.push({
    skuCode: newItem.skuCode,
    quantity: newItem.quantity,
    unitPrice: newItem.unitPrice ?? 0,
  });
  newItem.skuCode = '';
  newItem.quantity = null;
  newItem.unitPrice = null;
}

function removeItem(index: number) {
  items.splice(index, 1);
}

const itemColumns = [
  { title: $t('page.salesOrder.skuCode'), dataIndex: 'skuCode' },
  { title: $t('page.salesOrder.quantity'), dataIndex: 'quantity' },
  {
    title: $t('page.salesOrder.unitPrice'),
    dataIndex: 'unitPrice',
  },
  {
    title: $t('page.salesOrder.amount'),
    dataIndex: 'amount',
  },
];

const readonlyItemColumns = [
  { title: $t('page.salesOrder.skuCode'), dataIndex: 'skuCode' },
  { title: $t('page.salesOrder.quantity'), dataIndex: 'quantity' },
  {
    title: $t('page.salesOrder.fulfillQuantity'),
    dataIndex: 'fulfilledQuantity',
  },
  {
    title: $t('page.salesOrder.unitPrice'),
    dataIndex: 'unitPrice',
  },
  {
    title: $t('page.salesOrder.amount'),
    dataIndex: 'amount',
  },
  { title: $t('ui.table.action'), key: 'op', width: 90 },
];

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <div class="flex flex-col gap-4">
      <BaseForm />

      <div>
        <div class="mb-2 text-sm font-medium">
          {{ $t('page.salesOrder.items') }}
        </div>

        <template v-if="isCreate || isDraft">
          <div class="mb-2 flex gap-2">
            <a-input
              v-model:value="newItem.skuCode"
              :placeholder="$t('page.salesOrder.skuCode')"
              style="width: 40%"
            />
            <a-input-number
              v-model:value="newItem.quantity"
              :min="1"
              :placeholder="$t('page.salesOrder.quantity')"
              style="width: 25%"
            />
            <a-input-number
              v-model:value="newItem.unitPrice"
              :min="0"
              :placeholder="$t('page.salesOrder.unitPriceCents')"
              style="width: 25%"
            />
            <a-button type="primary" @click="addItem">
              {{ $t('page.salesOrder.addItem') }}
            </a-button>
          </div>
          <a-table
            v-if="items.length > 0"
            :columns="itemColumns"
            :data-source="items"
            :pagination="false"
            size="small"
            :row-key="(_r: any, i: number) => i"
          >
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.dataIndex === 'unitPrice'">
                {{ centsToYuan(record.unitPrice) }}
              </template>
              <template v-else-if="column.dataIndex === 'amount'">
                {{ centsToYuan(record.quantity * record.unitPrice) }}
              </template>
              <template v-else-if="column.key === 'op'">
                <a-button danger size="small" @click="removeItem(index)">
                  {{ $t('ui.button.delete') }}
                </a-button>
              </template>
            </template>
          </a-table>
        </template>

        <template v-else>
          <a-table
            :columns="readonlyItemColumns"
            :data-source="data?.row?.items ?? []"
            :pagination="false"
            size="small"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.dataIndex === 'unitPrice'">
                {{ centsToYuan(record.unitPrice) }}
              </template>
              <template v-else-if="column.dataIndex === 'amount'">
                {{ centsToYuan(record.amount) }}
              </template>
              <template v-else-if="column.key === 'op'">
                <a-button
                  v-if="isReturnable && (record.fulfilledQuantity ?? 0) > 0"
                  size="small"
                  @click="handleReturn(record)"
                >
                  {{ $t('page.stockPicking.return.salesTitle') }}
                </a-button>
              </template>
            </template>
          </a-table>
        </template>
      </div>

      <div v-if="!isCreate" class="flex justify-end gap-2">
        <template v-if="data?.row?.status === 'DRAFT'">
          <a-button @click="handleAction('cancel')">
            {{ $t('page.salesOrder.button.cancel') }}
          </a-button>
          <a-button type="primary" @click="handleAction('submit')">
            {{ $t('page.salesOrder.button.submit') }}
          </a-button>
        </template>
        <template v-else-if="data?.row?.status === 'SUBMITTED'">
          <a-button danger @click="handleAction('reject')">
            {{ $t('page.salesOrder.button.reject') }}
          </a-button>
          <a-button @click="handleAction('cancel')">
            {{ $t('page.salesOrder.button.cancel') }}
          </a-button>
          <a-button type="primary" @click="handleAction('approve')">
            {{ $t('page.salesOrder.button.approve') }}
          </a-button>
        </template>
        <template v-else-if="data?.row?.status === 'APPROVED'">
          <a-button @click="handleAction('cancel')">
            {{ $t('page.salesOrder.button.cancel') }}
          </a-button>
          <a-button type="primary" @click="handleAction('complete')">
            {{ $t('page.salesOrder.button.complete') }}
          </a-button>
        </template>
      </div>
    </div>
    <SalesReturnModal />
  </Drawer>
</template>
