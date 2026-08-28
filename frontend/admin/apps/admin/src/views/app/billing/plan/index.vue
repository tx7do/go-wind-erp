<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { apiClient, centsToYuanMonthly, limitLabel } from '#/api';

defineOptions({ name: 'BillingPlanManagement' });

const loading = ref(true);
const plans = ref<any[]>([]);

async function load() {
  loading.value = true;
  try {
    const resp = await apiClient.planAdminService.List({ noPaging: true } as any);
    plans.value = resp?.items ?? [];
  } catch {
    notification.error({ message: $t('ui.notification.load_failed') });
  } finally {
    loading.value = false;
  }
}

onMounted(load);

const columns = [
  { title: $t('page.billing.code'), dataIndex: 'code' },
  { title: $t('page.billing.name'), dataIndex: 'name' },
  {
    title: $t('page.billing.price'),
    dataIndex: 'priceCents',
  },
  {
    title: $t('page.billing.maxUsers'),
    dataIndex: 'maxUsers',
  },
  {
    title: $t('page.billing.maxOrders'),
    dataIndex: 'maxOrdersMonthly',
  },
  { title: $t('ui.table.createdAt'), dataIndex: 'createdAt' },
];
</script>

<template>
  <Page :title="$t('page.billing.planManagement')">
    <div class="p-2">
      <a-table
        :columns="columns"
        :data-source="plans"
        :loading="loading"
        :pagination="false"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'priceCents'">
            {{ centsToYuanMonthly(record.priceCents) }}
          </template>
          <template v-else-if="column.dataIndex === 'maxUsers'">
            {{ limitLabel(record.maxUsers) }}
          </template>
          <template v-else-if="column.dataIndex === 'maxOrdersMonthly'">
            {{ limitLabel(record.maxOrdersMonthly) }}
          </template>
        </template>
      </a-table>
    </div>
  </Page>
</template>
