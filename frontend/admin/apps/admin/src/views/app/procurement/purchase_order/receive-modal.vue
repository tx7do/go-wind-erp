<script lang="ts" setup>
import { ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  fetchListStockPickings,
  fetchListWarehouses,
  PaginationQuery,
} from '#/api';
import { apiClient } from '#/api/client';

const data = ref();

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
        // biome-ignore lint/style/noNonNullAssignment: 收货拣货单按 PO 过滤
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

      await apiClient.stockPickingService.Validate({ id: picking.id });

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
      baseFormApi.setValues({ quantity: data.value?.remaining });
      setLoading(false);
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
  </Modal>
</template>
