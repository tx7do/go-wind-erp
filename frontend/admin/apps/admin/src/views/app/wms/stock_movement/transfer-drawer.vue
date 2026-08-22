<script lang="ts" setup>
import { computed } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { useCreateStockPicking } from '#/api';

const getTitle = computed(() => $t('page.stockPicking.transfer.title'));

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
      label: $t('page.stockPicking.transfer.fromWarehouse'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const { fetchListWarehouses, PaginationQuery } = await import('#/api');
          const result = await fetchListWarehouses(
            // biome-ignore lint/style/noNonNullAssertion: 仓库下拉不分页
            new PaginationQuery({
              paging: { page: 1, pageSize: 500 },
            }),
          );
          return result?.items ?? [];
        },
        afterFetch: (warehouses: { code?: string; enable?: boolean }[]) =>
          warehouses
            .filter((w) => w.enable)
            .map((w) => ({ label: w.code, value: w.code })),
      },
    },
    {
      component: 'ApiSelect',
      fieldName: 'toWarehouseCode',
      label: $t('page.stockPicking.transfer.toWarehouse'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const { fetchListWarehouses, PaginationQuery } = await import('#/api');
          const result = await fetchListWarehouses(
            // biome-ignore lint/style/noNonNullAssertion: 仓库下拉不分页
            new PaginationQuery({
              paging: { page: 1, pageSize: 500 },
            }),
          );
          return result?.items ?? [];
        },
        afterFetch: (warehouses: { code?: string; enable?: boolean }[]) =>
          warehouses
            .filter((w) => w.enable)
            .map((w) => ({ label: w.code, value: w.code })),
      },
    },
    {
      component: 'Input',
      fieldName: 'productCode',
      label: $t('page.stockPicking.productCode'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'InputNumber',
      fieldName: 'plannedQuantity',
      label: $t('page.stockPicking.plannedQuantity'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Textarea',
      fieldName: 'remark',
      label: $t('page.stockPicking.transfer.remark'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
});

const createMutation = useCreateStockPicking();

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

    const fromWh = values.fromWarehouseCode as string | undefined;
    const toWh = values.toWarehouseCode as string | undefined;

    if (fromWh && toWh && fromWh === toWh) {
      notification.error({
        message: $t('page.stockPicking.transfer.sameWarehouse'),
      });
      setLoading(false);
      return;
    }

    try {
      await createMutation.mutateAsync({
        pickingType: 'INTERNAL',
        fromWarehouseCode: fromWh,
        toWarehouseCode: toWh,
        moves: [
          {
            productCode: values.productCode,
            plannedQuantity: values.plannedQuantity,
          },
        ],
        remark: values.remark,
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
