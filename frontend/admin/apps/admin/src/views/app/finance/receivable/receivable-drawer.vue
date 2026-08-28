<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import type { financeservicev1_Receipt_Method } from '#/api/generated/admin/service/v1';

import { useVbenForm } from '#/adapter/form';
import {
  apiClient,
  centsToYuan,
  receivableStatusToName,
  receiptMethodList,
} from '#/api';

const data = ref();
const receiptAmount = ref<null | number>(null);
const receiptMethod = ref('BANK_TRANSFER');
const acting = ref(false);

const isCreate = computed(() => data.value?.create);
const receivableStatus = computed(() => data.value?.row?.status);
const canPay = computed(
  () => receivableStatus.value === 'PENDING' || receivableStatus.value === 'PARTIAL',
);
const canCancel = computed(
  () =>
    receivableStatus.value === 'PENDING' && (data.value?.row?.paidAmount ?? 0) === 0,
);

const getTitle = computed(() =>
  isCreate.value
    ? $t('ui.modal.create', { moduleName: $t('page.receivable.moduleName') })
    : `${data.value?.row?.receivableNumber ?? ''} — ${receivableStatusToName(
        receivableStatus.value,
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
      fieldName: 'customerCode',
      label: $t('page.receivable.customerCode'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'InputNumber',
      fieldName: 'amountCents',
      label: $t('page.receivable.amountCents'),
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
      await apiClient.receivableService.Create({
        data: {
          customerCode: values.customerCode,
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
      receiptAmount.value = null;
      receiptMethod.value = 'BANK_TRANSFER';
      setLoading(false);
    }
  },
});

async function handleReceipt() {
  const amount = receiptAmount.value;
  if (!amount || amount <= 0) {
    notification.warning({ message: $t('page.receivable.receiptInvalid') });
    return;
  }
  acting.value = true;
  try {
    await apiClient.receiptService.Create({
      data: {
        receivableId: data.value.row.id,
        amount,
        method: receiptMethod.value as financeservicev1_Receipt_Method,
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
    await apiClient.receivableService.Cancel({ id: data.value.row.id });
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
          <a-descriptions-item :label="$t('page.receivable.soRef')">
            {{ data?.row?.soRef || '-' }}
          </a-descriptions-item>
          <a-descriptions-item :label="$t('page.receivable.customerCode')">
            {{ data?.row?.customerCode }}
          </a-descriptions-item>
          <a-descriptions-item :label="$t('page.receivable.amount')">
            {{ centsToYuan(data?.row?.amount) }}
          </a-descriptions-item>
          <a-descriptions-item :label="$t('page.receivable.paidAmount')">
            {{ centsToYuan(data?.row?.paidAmount) }}
          </a-descriptions-item>
        </a-descriptions>

        <!-- 收款（PENDING/PARTIAL） -->
        <div v-if="canPay" class="flex flex-col gap-2">
          <div class="text-sm font-medium">
            {{ $t('page.receivable.recordReceipt') }}
          </div>
          <div class="flex gap-2">
            <a-input-number
              v-model:value="receiptAmount"
              :min="1"
              :placeholder="$t('page.receivable.receiptAmountCents')"
              style="width: 45%"
            />
            <a-select
              v-model:value="receiptMethod"
              :options="receiptMethodList"
              style="width: 35%"
            />
            <a-button
              type="primary"
              :loading="acting"
              style="width: 20%"
              @click="handleReceipt"
            >
              {{ $t('page.receivable.button.receipt') }}
            </a-button>
          </div>
        </div>

        <div class="flex justify-end gap-2">
          <a-button v-if="canCancel" :loading="acting" @click="handleCancel">
            {{ $t('page.receivable.button.cancel') }}
          </a-button>
        </div>
      </template>
    </div>
  </Drawer>
</template>
