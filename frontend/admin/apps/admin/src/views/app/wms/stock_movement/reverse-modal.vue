<script lang="ts" setup>
import { ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { useReverseStockMovement } from '#/api';

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
      component: 'Textarea',
      fieldName: 'reason',
      label: $t('page.stockMovement.reverse.reason'),
      componentProps: {
        placeholder: $t('page.stockMovement.reverse.reasonHint'),
        allowClear: true,
      },
      rules: 'required',
    },
  ],
});

const reverseMutation = useReverseStockMovement();

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

    try {
      await reverseMutation.mutateAsync({
        id: data.value?.id,
        reason: values.reason as string | undefined,
      });

      notification.success({
        message: $t('ui.notification.create_success'),
      });
    } catch {
      notification.error({
        message: $t('ui.notification.create_failed'),
      });
    } finally {
      modalApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      data.value = modalApi.getData<any>();
      setLoading(false);
    }
  },
});

function setLoading(loading: boolean) {
  modalApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Modal :title="$t('page.stockMovement.reverse.title')">
    <BaseForm />
  </Modal>
</template>
