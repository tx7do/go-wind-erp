<script lang="ts" setup>
import { computed } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { fetchListWarehouses, PaginationQuery, useTransferStock } from '#/api';

const getTitle = computed(() => $t('page.stockMovement.transfer.title'));

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
      fieldName: 'fromWarehouseCode',
      label: $t('page.stockMovement.transfer.fromWarehouse'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await fetchListWarehouses(
            // biome-ignore lint/style/noNonNullAssertion: 调拨仓库下拉不分页
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
      component: 'ApiSelect',
      fieldName: 'toWarehouseCode',
      label: $t('page.stockMovement.transfer.toWarehouse'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await fetchListWarehouses(
            // biome-ignore lint/style/noNonNullAssertion: 调拨仓库下拉不分页
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
      component: 'Input',
      fieldName: 'skuCode',
      label: $t('page.stockMovement.skuCode'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'InputNumber',
      fieldName: 'quantity',
      label: $t('page.stockMovement.transfer.quantity'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Textarea',
      fieldName: 'remark',
      label: $t('page.stockMovement.transfer.remark'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
});

const transferMutation = useTransferStock();

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
    const from = values.fromWarehouseCode as string | undefined;
    const to = values.toWarehouseCode as string | undefined;

    if (from && to && from === to) {
      notification.error({
        message: $t('page.stockMovement.transfer.sameWarehouse'),
      });
      setLoading(false);
      return;
    }

    try {
      await transferMutation.mutateAsync({
        fromWarehouseCode: from,
        toWarehouseCode: to,
        skuCode: values.skuCode as string | undefined,
        quantity: values.quantity as number | undefined,
        remark: values.remark as string | undefined,
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
