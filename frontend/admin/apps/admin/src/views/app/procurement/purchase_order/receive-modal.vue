<script lang="ts" setup>
import { ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { apiClient, fetchListWarehouses, PaginationQuery } from '#/api';

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

    try {
      // 收货 = 一笔带 poId 的 INBOUND 流水：服务端 ApplyReceipt 累计收货量
      // （防超收守卫）并回写库存，全收后自动完结 PO。
      await apiClient.stockMovementService.Create({
        data: {
          warehouseCode: values.warehouseCode as string | undefined,
          skuCode: data.value?.skuCode as string | undefined,
          delta: qty,
          movementType: 'INBOUND',
          poId: data.value?.poId as number | undefined,
        },
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
