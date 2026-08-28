<script lang="ts" setup>
import { computed, reactive, ref } from 'vue';

import { useVbenDrawer, useVbenModal } from '@vben/common-ui';

import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import type { procurementservicev1_PurchaseOrderItem } from '#/api/generated/admin/service/v1';

import { useVbenForm } from '#/adapter/form';
import {
  apiClient,
  centsToYuan,
  fetchListSuppliers,
  fetchListWarehouses,
  purchaseOrderStatusToName,
} from '#/api';

import PurchaseReturnModalComponent from './purchase-return-modal.vue';
import ReceiveModalComponent from './receive-modal.vue';

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
// APPROVED 状态下可按明细收货（服务端 ApplyReceipt 有审批闸门与防超收守卫）。
const isApproved = computed(() => data.value?.row?.status === 'APPROVED');
// APPROVED/COMPLETED 状态下可按明细退货（服务端 ApplyReceiptReturnTx 有
// 状态门与防超退守卫，前端按钮按已收数量显隐）。
const isReturnable = computed(
  () =>
    data.value?.row?.status === 'APPROVED' ||
    data.value?.row?.status === 'COMPLETED',
);

// 收货弹窗：收集仓库+数量，成功后关闭抽屉让列表刷新（与工作流动作同模式）。
const [ReceiveModal, receiveModalApi] = useVbenModal({
  connectedComponent: ReceiveModalComponent,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      drawerApi.close();
    }
  },
});

function handleReceive(record: any) {
  receiveModalApi.setData({
    poId: data.value.row.id,
    skuCode: record.skuCode,
    remaining: (record.quantity ?? 0) - (record.receivedQuantity ?? 0),
  });
  receiveModalApi.open();
}

// 退货弹窗：收集退货数量，成功后关闭抽屉让列表刷新。
const [PurchaseReturnModal, purchaseReturnModalApi] = useVbenModal({
  connectedComponent: PurchaseReturnModalComponent,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      drawerApi.close();
    }
  },
});

function handleReturn(record: any) {
  purchaseReturnModalApi.setData({
    poId: data.value.row.id,
    poItemId: record.id,
    returnable: record.receivedQuantity ?? 0,
  });
  purchaseReturnModalApi.open();
}

const getTitle = computed(() =>
  isCreate.value
    ? $t('ui.modal.create', {
        moduleName: $t('page.purchaseOrder.moduleName'),
      })
    : `${data.value?.row?.poNumber ?? ''} — ${purchaseOrderStatusToName(
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
      fieldName: 'supplierCode',
      label: $t('page.purchaseOrder.supplierCode'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await fetchListSuppliers(
            // biome-ignore lint/style/noNonNullAssertion: 供应商下拉不分页
            new (await import('#/transport/rest')).PaginationQuery({
              paging: { page: 1, pageSize: 500 },
            }),
          );
          return result?.items ?? [];
        },
        afterFetch: (
          suppliers: {
            code?: string;
            enable?: boolean;
            name?: string;
          }[],
        ) =>
          suppliers
            .filter((s) => s.enable)
            .map((s) => ({ label: `${s.code} — ${s.name}`, value: s.code })),
      },
    },

    {
      component: 'ApiSelect',
      fieldName: 'warehouseCode',
      label: $t('page.purchaseOrder.warehouseCode'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await fetchListWarehouses(
            // biome-ignore lint/style/noNonNullAssertion: 仓库下拉不分页
            new (await import('#/transport/rest')).PaginationQuery({
              paging: { page: 1, pageSize: 500 },
            }),
          );
          return result?.items ?? [];
        },
        afterFetch: (
          warehouses: {
            code?: string;
            enable?: boolean;
            name?: string;
          }[],
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
        (i): procurementservicev1_PurchaseOrderItem => ({
          skuCode: i.skuCode,
          quantity: i.quantity as number,
          unitPrice: i.unitPrice ?? 0,
        }),
      );
    if (payloadItems.length === 0) {
      notification.warning({
        message: $t('page.purchaseOrder.itemsRequired'),
      });
      return;
    }

    setLoading(true);

    try {
      await (data.value?.create
        ? apiClient.purchaseOrderService.Create({
            data: {
              supplierCode: values.supplierCode,
              warehouseCode: values.warehouseCode,
              remark: values.remark,
              items: payloadItems,
            },
          })
        : apiClient.purchaseOrderService.Update({
            id: data.value.row.id,
            data: {
              supplierCode: values.supplierCode,
              warehouseCode: values.warehouseCode,
              remark: values.remark,
              items: payloadItems,
            },
            updateMask: 'supplierCode,warehouseCode,remark,items',
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
      // 供只读明细表/收货/退货按钮使用。
      if (!data.value?.create && data.value?.row?.id) {
        try {
          const full = await apiClient.purchaseOrderService.Get({
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
        supplierCode: data.value?.row?.supplierCode,
        warehouseCode: data.value?.row?.warehouseCode,
        remark: data.value?.row?.remark,
      });

      setLoading(false);
    }
  },
});

// 动作（仅非新建抽屉显示；按当前状态决定可用性，服务端仍是权威守卫）。
async function handleAction(
  action: 'approve' | 'cancel' | 'complete' | 'reject' | 'submit',
) {
  setLoading(true);
  try {
    const id = data.value.row.id;
    if (action === 'submit') {
      await apiClient.purchaseOrderService.Submit({ id });
    } else if (action === 'approve') {
      await apiClient.purchaseOrderService.Approve({ id });
    } else if (action === 'reject') {
      await apiClient.purchaseOrderService.Reject({ id });
    } else if (action === 'cancel') {
      await apiClient.purchaseOrderService.Cancel({ id });
    } else {
      await apiClient.purchaseOrderService.Complete({ id });
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
    notification.warning({ message: $t('page.purchaseOrder.itemInvalid') });
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
  { title: $t('page.purchaseOrder.skuCode'), dataIndex: 'skuCode' },
  { title: $t('page.purchaseOrder.quantity'), dataIndex: 'quantity' },
  {
    title: $t('page.purchaseOrder.unitPrice'),
    dataIndex: 'unitPrice',
  },
  {
    title: $t('page.purchaseOrder.amount'),
    dataIndex: 'amount',
  },
];

// 只读明细列：额外展示已收数量与收货操作（收货仅 APPROVED 且未收满时可用）。
// 退货（APPROVED/COMPLETED 且已收>0）走 purchase-return-modal。
const readonlyItemColumns = [
  { title: $t('page.purchaseOrder.skuCode'), dataIndex: 'skuCode' },
  { title: $t('page.purchaseOrder.quantity'), dataIndex: 'quantity' },
  {
    title: $t('page.purchaseOrder.receivedQuantity'),
    dataIndex: 'receivedQuantity',
  },
  {
    title: $t('page.purchaseOrder.unitPrice'),
    dataIndex: 'unitPrice',
  },
  {
    title: $t('page.purchaseOrder.amount'),
    dataIndex: 'amount',
  },
  { title: $t('ui.table.action'), key: 'op', width: 140 },
];

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <div class="flex flex-col gap-4">
      <BaseForm />

      <!-- 明细编辑（新建/DRAFT 可改；其他状态只读展示已有明细） -->
      <div>
        <div class="mb-2 text-sm font-medium">
          {{ $t('page.purchaseOrder.items') }}
        </div>

        <template v-if="isCreate || isDraft">
          <div class="mb-2 flex gap-2">
            <a-input
              v-model:value="newItem.skuCode"
              :placeholder="$t('page.purchaseOrder.skuCode')"
              style="width: 40%"
            />
            <a-input-number
              v-model:value="newItem.quantity"
              :min="1"
              :placeholder="$t('page.purchaseOrder.quantity')"
              style="width: 25%"
            />
            <a-input-number
              v-model:value="newItem.unitPrice"
              :min="0"
              :placeholder="$t('page.purchaseOrder.unitPriceCents')"
              style="width: 25%"
            />
            <a-button type="primary" @click="addItem">
              {{ $t('page.purchaseOrder.addItem') }}
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
              <template v-else-if="column.dataIndex === 'action'"> </template>
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
                <div class="flex gap-1">
                  <a-button
                    v-if="
                      isApproved &&
                      (record.quantity ?? 0) - (record.receivedQuantity ?? 0) > 0
                    "
                    size="small"
                    type="primary"
                    @click="handleReceive(record)"
                  >
                    {{ $t('page.purchaseOrder.button.receive') }}
                  </a-button>
                  <a-button
                    v-if="isReturnable && (record.receivedQuantity ?? 0) > 0"
                    size="small"
                    @click="handleReturn(record)"
                  >
                    {{ $t('page.stockPicking.return.purchaseTitle') }}
                  </a-button>
                  <a-tag
                    v-else-if="isApproved"
                    color="green"
                    class="ml-1 self-center"
                  >
                    {{ record.receivedQuantity ?? 0 }}/{{ record.quantity ?? 0 }}
                  </a-tag>
                </div>
              </template>
            </template>
          </a-table>
        </template>
      </div>

      <!-- 动作区（非新建） -->
      <div v-if="!isCreate" class="flex justify-end gap-2">
        <template v-if="data?.row?.status === 'DRAFT'">
          <a-button @click="handleAction('cancel')">
            {{ $t('page.purchaseOrder.button.cancel') }}
          </a-button>
          <a-button type="primary" @click="handleAction('submit')">
            {{ $t('page.purchaseOrder.button.submit') }}
          </a-button>
        </template>
        <template v-else-if="data?.row?.status === 'SUBMITTED'">
          <a-button danger @click="handleAction('reject')">
            {{ $t('page.purchaseOrder.button.reject') }}
          </a-button>
          <a-button @click="handleAction('cancel')">
            {{ $t('page.purchaseOrder.button.cancel') }}
          </a-button>
          <a-button type="primary" @click="handleAction('approve')">
            {{ $t('page.purchaseOrder.button.approve') }}
          </a-button>
        </template>
        <template v-else-if="data?.row?.status === 'APPROVED'">
          <a-button @click="handleAction('cancel')">
            {{ $t('page.purchaseOrder.button.cancel') }}
          </a-button>
          <a-button type="primary" @click="handleAction('complete')">
            {{ $t('page.purchaseOrder.button.complete') }}
          </a-button>
        </template>
      </div>
    </div>
    <ReceiveModal />
    <PurchaseReturnModal />
  </Drawer>
</template>
