<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { apiClient, enableBoolList, makeUpdateMask } from '#/api';

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.customer.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.customer.moduleName') }),
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
      fieldName: 'code',
      label: $t('page.customer.code'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.customer.name'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'Input',
      fieldName: 'contact',
      label: $t('page.customer.contact'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'Input',
      fieldName: 'phone',
      label: $t('page.customer.phone'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'InputNumber',
      fieldName: 'creditLimitYuan',
      label: $t('page.customer.creditLimit'),
      componentProps: {
        placeholder: $t('page.customer.creditLimitHint'),
        allowClear: true,
        min: 0,
        precision: 2,
      },
    },

    {
      component: 'RadioGroup',
      fieldName: 'enable',
      defaultValue: true,
      label: $t('page.customer.enable'),
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap',
        options: enableBoolList,
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
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    setLoading(true);

    const values = await baseFormApi.getValues();

    // 信用额度：界面元 → 存储分；空 = 不限（0）。
    const { creditLimitYuan, ...rest } = values as Record<string, any>;
    const payload = {
      ...rest,
      creditLimit: Math.round((Number(creditLimitYuan) || 0) * 100),
    };

    try {
      await (data.value?.create
        ? apiClient.customerService.Create({ data: { ...payload } })
        : apiClient.customerService.Update({
            id: data.value.row.id,
            data: { ...payload },
            updateMask: makeUpdateMask(Object.keys(payload)),
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

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      data.value = drawerApi.getData<Record<string, any>>();

      baseFormApi.setValues({
        ...data.value?.row,
        creditLimitYuan:
          data.value?.row?.creditLimit != null
            ? data.value.row.creditLimit / 100
            : undefined,
      });

      setLoading(false);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <BaseForm />
  </Drawer>
</template>
