<script lang="ts" setup>
import type {
  procurementservicev1_PurchaseOrder,
} from '#/api/generated/admin/service/v1';

import { ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  apiClient,
  fetchListStockPickings,
  fetchListWarehouses,
  PaginationQuery,
} from '#/api';

interface LotRow {
  skuCode: string;
  quantity: null | number;
  lotName: string;
  expiry: undefined | string;
}

const data = ref();
// 收货明细行（批次录入）：SKU × 批次号 × 效期；效期为空 = 不限期。
const lotRows = ref<LotRow[]>([]);

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
      fieldName: 'warehouseCode',
      label: $t('page.purchaseOrder.receive.warehouse'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await fetchListWarehouses(
            // biome-ignore lint/style/noNonNullAssertion: 收货仓库下拉不分页
            new PaginationQuery({
              paging: { page: 1, pageSize: 500 },
            }),
          );
          return result?.items ?? [];
        },
        afterFetch: (
          warehouses: {
            code?: string;
            enable?: boolean;
          }[],
        ) =>
          warehouses
            .filter((w) => w.enable)
            .map((w) => ({ label: w.code, value: w.code })),
      },
    },
    {
      component: 'InputNumber',
      fieldName: 'quantity',
      label: $t('page.purchaseOrder.receive.quantity'),
      rules: 'required',
      componentProps: {
        placeholder: $t('page.purchaseOrder.receive.quantityHint'),
        allowClear: true,
        min: 1,
      },
    },
  ],
});

async function loadOrderItems(poId?: number) {
  lotRows.value = [];
  if (!poId) return;
  try {
    const po:
      | procurementservicev1_PurchaseOrder
      | undefined = await apiClient.purchaseOrderService.Get({ id: poId });
    lotRows.value = (po?.items ?? [])
      .filter((it) => (it.quantity ?? 0) - (it.receivedQuantity ?? 0) > 0)
      .map((it) => ({
        skuCode: it.skuCode ?? '',
        quantity: (it.quantity ?? 0) - (it.receivedQuantity ?? 0),
        lotName: '',
        expiry: undefined,
      }));
  } catch {
    // 拉取失败时退化为无批次录入（批次为可选项）
  }
}

const [Modal, modalApi] = useVbenModal({
  onCancel() {
    modalApi.close();
  },

  async onConfirm() {
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    setLoading(true);

    const values = await baseFormApi.getValues();
    const qty = values.quantity as number | undefined;
    const remaining = (data.value?.remaining ?? 0) as number;

    if (qty && (qty <= 0 || qty > remaining)) {
      notification.error({
        message: $t('page.purchaseOrder.receive.exceedsRemaining'),
      });
      setLoading(false);
      return;
    }

    const poId = data.value?.poId as number | undefined;

    try {
      // 入库拣货单在 PO 审批时由服务端自动创建；此处通过 purchaseOrderId
      // 关联到该拣货单，调用 Validate 推进收货流程。
      const pickingList = await fetchListStockPickings(
        // biome-ignore lint/noNonNullAssertion: 收货拣货单按 PO 过滤
        new PaginationQuery({
          paging: { page: 1, pageSize: 50 },
          formValues: { purchaseOrderId: poId },
        }),
      );

      const picking = (pickingList?.items ?? []).find(
        (p) => p.purchaseOrderId === poId,
      );

      if (!picking || picking.id === undefined) {
        notification.error({
          message: $t('ui.notification.operation_failed'),
        });
        return;
      }

      // 入库拣货单在 PO 审批时创建，初始派生态为 DRAFT。Validate 要求
      // CONFIRMED 态，因此 DRAFT 时先调 Confirm（借鉴 Odoo action_confirm），
      // 再调 Validate（借鉴 Odoo button_validate / _action_done）。
      if (picking.derivedState === 'DRAFT') {
        await apiClient.stockPickingService.Confirm({ id: picking.id });
      }

      // 批次指派：仅收集填写了批次号的行（效期可选）。
      const lotAssignments = lotRows.value
        .filter((r) => r.skuCode && r.lotName.trim() !== '')
        .map((r) => ({
          productCode: r.skuCode,
          lotName: r.lotName.trim(),
          expiryDate: r.expiry,
        }));

      await apiClient.stockPickingService.Validate({
        id: picking.id,
        lotAssignments,
      });

      notification.success({
        message: $t('ui.notification.operation_success'),
      });
    } catch {
      notification.error({
        message: $t('ui.notification.operation_failed'),
      });
    } finally {
      modalApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      data.value = modalApi.getData<any>();
      baseFormApi.setValues({
        quantity: data.value?.remaining,
        // 预填 PO 收货仓库（仍可改选）
        warehouseCode: data.value?.warehouseCode,
      });
      setLoading(false);
      loadOrderItems(data.value?.poId);
    }
  },
});

function setLoading(loading: boolean) {
  modalApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Modal :title="$t('page.purchaseOrder.receive.title')">
    <BaseForm />

    <div v-if="lotRows.length > 0" class="mt-2">
      <div class="mb-2 text-sm font-medium">
        {{ $t('page.purchaseOrder.receive.lotSection') }}
      </div>
      <div class="mb-2 text-xs text-gray-400">
        {{ $t('page.purchaseOrder.receive.lotHint') }}
      </div>
      <div
        v-for="row in lotRows"
        :key="row.skuCode"
        class="mb-2 flex items-center gap-2"
      >
        <span class="w-32 truncate">
          {{ row.skuCode }} × {{ row.quantity }}
        </span>
        <a-input
          v-model:value="row.lotName"
          :placeholder="$t('page.purchaseOrder.receive.lotName')"
          style="width: 35%"
        />
        <a-date-picker
          :placeholder="$t('page.purchaseOrder.receive.expiryDate')"
          style="width: 35%"
          value-format="YYYY-MM-DDT00:00:00Z"
          @change="(_d: any, iso: string) => (row.expiry = iso || undefined)"
        />
      </div>
    </div>
  </Modal>
</template>
