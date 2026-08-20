<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { apiClient, approvalStatusToName } from '#/api';

const data = ref();
const comment = ref('');
const acting = ref(false);

const isCreate = computed(() => data.value?.mode === 'create');
const isPending = computed(() => data.value?.row?.status === 'PENDING');

const getTitle = computed(() =>
  isCreate.value
    ? $t('ui.modal.create', { moduleName: $t('page.approval.moduleName') })
    : `${$t('page.approval.moduleName')} — ${approvalStatusToName(
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
      component: 'Input',
      fieldName: 'title',
      label: $t('page.approval.title'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'Input',
      fieldName: 'bizType',
      label: $t('page.approval.bizType'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'Input',
      fieldName: 'bizRef',
      label: $t('page.approval.bizRef'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'Textarea',
      fieldName: 'summary',
      label: $t('page.approval.summary'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
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
      await apiClient.approvalRequestService.Create({ data: { ...values } });
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
      comment.value = data.value?.row?.comment ?? '';
      setLoading(false);
    }
  },
});

async function handleAction(action: 'approve' | 'reject' | 'cancel') {
  acting.value = true;
  try {
    if (action === 'approve') {
      await apiClient.approvalRequestService.Approve({
        id: data.value.row.id,
        comment: comment.value || undefined,
      });
    } else if (action === 'reject') {
      await apiClient.approvalRequestService.Reject({
        id: data.value.row.id,
        comment: comment.value || undefined,
      });
    } else {
      await apiClient.approvalRequestService.Cancel({
        id: data.value.row.id,
      });
    }

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
    <BaseForm v-if="isCreate" />

    <div v-else class="flex flex-col gap-4">
      <a-descriptions :column="1" bordered size="small">
        <a-descriptions-item :label="$t('page.approval.title')">
          {{ data?.row?.title }}
        </a-descriptions-item>
        <a-descriptions-item :label="$t('page.approval.bizType')">
          {{ data?.row?.bizType }}
        </a-descriptions-item>
        <a-descriptions-item :label="$t('page.approval.bizRef')">
          {{ data?.row?.bizRef }}
        </a-descriptions-item>
        <a-descriptions-item :label="$t('page.approval.summary')">
          {{ data?.row?.summary }}
        </a-descriptions-item>
      </a-descriptions>

      <a-textarea
        v-model:value="comment"
        :placeholder="$t('page.approval.commentHint')"
        :rows="3"
        :disabled="!isPending"
      />

      <div v-if="isPending" class="flex justify-end gap-2">
        <a-button danger :loading="acting" @click="handleAction('reject')">
          {{ $t('page.approval.button.reject') }}
        </a-button>
        <a-button :loading="acting" @click="handleAction('cancel')">
          {{ $t('page.approval.button.cancelRequest') }}
        </a-button>
        <a-button type="primary" :loading="acting" @click="handleAction('approve')">
          {{ $t('page.approval.button.approve') }}
        </a-button>
      </div>
    </div>
  </Drawer>
</template>
