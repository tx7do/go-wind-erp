<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';

import { notification } from 'ant-design-vue';

import type { financeservicev1_MonthlyProfit } from '#/api/generated/admin/service/v1';

import { apiClient, centsToYuan } from '#/api';
import { $t } from '#/locales';

const loading = ref(true);
const items = ref<financeservicev1_MonthlyProfit[]>([]);

const columns = [
  {
    title: $t('page.profit.month'),
    dataIndex: 'month',
    key: 'month',
  },
  {
    title: $t('page.profit.revenue'),
    dataIndex: 'revenue',
    key: 'revenue',
    slots: { default: 'revenue' },
  },
  {
    title: $t('page.profit.cogs'),
    dataIndex: 'cogs',
    key: 'cogs',
    slots: { default: 'cogs' },
  },
  {
    title: $t('page.profit.profit'),
    dataIndex: 'profit',
    key: 'profit',
    slots: { default: 'profit' },
  },
];

async function load() {
  loading.value = true;
  try {
    const resp = await apiClient.financeReportService.ProfitReport({});
    items.value = resp.items ?? [];
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  load();
});
</script>

<template>
  <Page auto-content-height>
    <a-card :title="$t('page.profit.title')">
      <a-alert
        class="mb-3"
        type="warning"
        :message="$t('page.profit.disclaimer')"
        show-icon
      />
      <a-table
        :columns="columns"
        :data-source="items"
        :loading="loading"
        :pagination="false"
        row-key="month"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'revenue'">
            {{ centsToYuan(record.revenue) }}
          </template>
          <template v-else-if="column.key === 'cogs'">
            {{ centsToYuan(record.cogs) }}
          </template>
          <template v-else-if="column.key === 'profit'">
            {{ centsToYuan(record.profit) }}
          </template>
        </template>
      </a-table>
    </a-card>
  </Page>
</template>
