<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import type { financeservicev1_Payment_Method } from '#/api/generated/admin/service/v1';

import { useVbenForm } from '#/adapter/form';
import {
  apiClient,
  centsToYuan,
  payableStatusToName,
  paymentMethodList,
} from '#/api';

const data = ref();
const paymentAmount = ref<null | number>(null);
const paymentMethod = ref('BANK_TRANSFER');
const acting = ref(false);

const isCreate = computed(() => data.value?.create);
const payableStatus = computed(() => data.value?.row?.status);
const canPay = computed(
  () => payableStatus.value === 'PENDING' || payableStatus.value === 'PARTIAL',
);
const canCancel = computed(
  () =>
    payableStatus.value === 'PENDING' && (data.value?.row?.paidAmount ?? 0) === 0,
);

const getTitle = computed(() =>
  isCreate.value
    ? $t('ui.modal.create', { moduleName: $t('page.payable.moduleName') })
    : `${data.value?.row?.payableNumber ?? ''} — ${payableStatusToName(
        payableStatus.value,
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
      component: 'Input',
      fieldName: 'supplierCode',
      label: $t('page.payable.supplierCode'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'InputNumber',
      fieldName: 'amountCents',
      label: $t('page.payable.amountCents'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        min: 1,
        precision: 0,
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
    if (!isCreate.value) {
      drawerApi.close();
      return;
    }

    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    setLoading(true);
    const values = await baseFormApi.getValues();

    try {
      await apiClient.payableService.Create({
        data: {
          supplierCode: values.supplierCode,
          amount: values.amountCents,
          remark: values.remark,
        },
      });
      notification.success({
        message: $t('ui.notification.create_success'),
      });
    } catch {
      notification.error({
        message: $t('ui.notification.create_failed'),
      });
    } finally {
      drawerApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      data.value = drawerApi.getData<Record<string, any>>();
      paymentAmount.value = null;
      paymentMethod.value = 'BANK_TRANSFER';
      setLoading(false);
    }
  },
});

async function handlePay() {
  const amount = paymentAmount.value;
  if (!amount || amount <= 0) {
    notification.warning({ message: $t('page.payable.paymentInvalid') });
    return;
  }
  acting.value = true;
  try {
    await apiClient.paymentService.Create({
      data: {
        payableId: data.value.row.id,
        amount,
        method: paymentMethod.value as financeservicev1_Payment_Method,
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
    acting.value = false;
    drawerApi.close();
  }
}

async function handleCancel() {
  acting.value = true;
  try {
    await apiClient.payableService.Cancel({ id: data.value.row.id });
    notification.success({
      message: $t('ui.notification.operation_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  } finally {
    acting.value = false;
    drawerApi.close();
  }
}

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <div class="flex flex-col gap-4">
      <BaseForm v-if="isCreate" />

      <template v-else>
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item :label="$t('page.payable.poRef')">
            {{ data?.row?.poRef || '-' }}
          </a-descriptions-item>
          <a-descriptions-item :label="$t('page.payable.supplierCode')">
            {{ data?.row?.supplierCode }}
          </a-descriptions-item>
          <a-descriptions-item :label="$t('page.payable.amount')">
            {{ centsToYuan(data?.row?.amount) }}
          </a-descriptions-item>
          <a-descriptions-item :label="$t('page.payable.paidAmount')">
            {{ centsToYuan(data?.row?.paidAmount) }}
          </a-descriptions-item>
        </a-descriptions>

        <!-- 付款（PENDING/PARTIAL） -->
        <div v-if="canPay" class="flex flex-col gap-2">
          <div class="text-sm font-medium">
            {{ $t('page.payable.recordPayment') }}
          </div>
          <div class="flex gap-2">
            <a-input-number
              v-model:value="paymentAmount"
              :min="1"
              :placeholder="$t('page.payable.paymentAmountCents')"
              style="width: 45%"
            />
            <a-select
              v-model:value="paymentMethod"
              :options="paymentMethodList"
              style="width: 35%"
            />
            <a-button
              type="primary"
              :loading="acting"
              style="width: 20%"
              @click="handlePay"
            >
              {{ $t('page.payable.button.pay') }}
            </a-button>
          </div>
        </div>

        <div class="flex justify-end gap-2">
          <a-button v-if="canCancel" :loading="acting" @click="handleCancel">
            {{ $t('page.payable.button.cancel') }}
          </a-button>
        </div>
      </template>
    </div>
  </Drawer>
</template>
