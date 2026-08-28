<script lang="ts" setup>
import { ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { useCreateSalesReturn } from '#/api';

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
      component: 'InputNumber',
      fieldName: 'quantity',
      label: $t('page.stockPicking.return.quantity'),
      rules: 'required',
      componentProps: {
        placeholder: `${$t('page.stockPicking.return.returnable')}: {max}`,
        allowClear: true,
        min: 1,
      },
    },
  ],
});

const createReturnMutation = useCreateSalesReturn();

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
    const returnable = (data.value?.returnable ?? 0) as number;

    if (qty && (qty <= 0 || qty > returnable)) {
      notification.error({
        message: `${$t('page.stockPicking.return.quantity')} > ${$t('page.stockPicking.return.returnable')} (${returnable})`,
      });
      setLoading(false);
      return;
    }

    try {
      await createReturnMutation.mutateAsync({
        salesOrderId: data.value.soId,
        items: [
          {
            salesOrderItemId: data.value.soItemId,
            quantity: qty as number,
          },
        ],
      });

      notification.success({
        message: $t('page.stockPicking.return.salesSuccess'),
      });
    } catch {
      notification.error({
        message: $t('page.stockPicking.return.failed'),
      });
    } finally {
      modalApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      data.value = modalApi.getData<any>();
      baseFormApi.setValues({ quantity: data.value?.returnable });
      setLoading(false);
    }
  },
});

function setLoading(loading: boolean) {
  modalApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Modal :title="$t('page.stockPicking.return.salesTitle')">
    <BaseForm />
  </Modal>
</template>
