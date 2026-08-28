<script lang="ts" setup>
import { computed } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { useCreateStockPicking } from '#/api';

const getTitle = computed(() => $t('page.stockPicking.adjustment.title'));

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
      label: $t('page.stockPicking.adjustment.warehouse'),
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
      // 带符号差异数：正=盘盈（INVENTORY_LOSS→仓库），负=盘亏（仓库→INVENTORY_LOSS）
      component: 'InputNumber',
      fieldName: 'plannedQuantity',
      label: $t('page.stockPicking.adjustment.diff'),
      rules: 'required',
      componentProps: {
        placeholder: $t('page.stockPicking.adjustment.diffTip'),
        allowClear: true,
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
      component: 'Textarea',
      fieldName: 'remark',
      label: $t('page.stockPicking.adjustment.remark'),
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

    const diff = values.plannedQuantity as number | undefined;
    if (!diff || diff === 0) {
      notification.error({
        message: $t('page.stockPicking.adjustment.diffNonZero'),
      });
      setLoading(false);
      return;
    }

    try {
      await createMutation.mutateAsync({
        pickingType: 'INVENTORY_ADJUSTMENT',
        fromWarehouseCode: values.fromWarehouseCode,
        moves: [
          {
            productCode: values.productCode,
            plannedQuantity: diff,
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
