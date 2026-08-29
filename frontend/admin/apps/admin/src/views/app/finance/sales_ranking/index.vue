<script lang="ts" setup>
import { onMounted, reactive, ref, shallowRef, watch } from 'vue';

import { Page } from '@vben/common-ui';

import type { EchartsUIType } from '@vben/plugins/echarts';

import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

import { notification } from 'ant-design-vue';

import { apiClient, centsToYuan } from '#/api';
import { $t } from '#/locales';
import { exportRowsToExcel } from '#/utils/export-excel';

defineOptions({ name: 'SalesRankingReport' });

const loading = ref(false);
const items = ref<any[]>([]);
const form = reactive({
  dimension: 'SKU',
  fromDate: undefined as string | undefined,
  toDate: undefined as string | undefined,
  limit: 10,
});

const dimensionOptions = [
  { value: 'SKU', label: $t('page.salesRanking.bySku') },
  { value: 'CUSTOMER', label: $t('page.salesRanking.byCustomer') },
];

const chartRef = shallowRef<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const columns = [
  { title: $t('page.salesRanking.rank'), key: 'rank', width: 70 },
  { title: $t('page.salesRanking.name'), dataIndex: 'label' },
  {
    title: $t('page.salesRanking.quantity'),
    dataIndex: 'quantity',
    width: 120,
    align: 'right' as const,
  },
  {
    title: $t('page.salesRanking.amount'),
    dataIndex: 'amount',
    width: 140,
    align: 'right' as const,
  },
];

async function load() {
  loading.value = true;
  try {
    const resp = await apiClient.financeReportService.GetSalesRanking({
      dimension: form.dimension,
      fromDate: form.fromDate
        ? new Date(`${form.fromDate}T00:00:00Z`)
        : undefined,
      toDate: form.toDate ? new Date(`${form.toDate}T23:59:59Z`) : undefined,
      limit: form.limit,
    } as any);
    items.value = resp?.items ?? [];
  } catch {
    notification.error({ message: $t('ui.notification.load_failed') });
  } finally {
    loading.value = false;
  }
}

onMounted(load);

watch(items, (val) => {
  if (!val.length) return;
  renderEcharts({
    grid: { top: 20, right: 30, bottom: 30, left: 120 },
    xAxis: { type: 'value' },
    yAxis: {
      type: 'category',
      inverse: true,
      data: val.map((i) => i.label ?? i.key),
    },
    series: [
      {
        type: 'bar',
        data: val.map((i) => Number(i.amount ?? 0) / 100),
        label: { show: true, position: 'right' },
      },
    ],
  });
});

/** 导出 Excel（排名/名称/数量/金额）。 */
function handleExport() {
  exportRowsToExcel(
    `销售排行_${form.dimension}_${new Date().toISOString().slice(0, 10)}`,
    '销售排行',
    [
      $t('page.salesRanking.rank'),
      $t('page.salesRanking.name'),
      $t('page.salesRanking.quantity'),
      $t('page.salesRanking.amount'),
    ],
    items.value.map((it, idx) => [
      idx + 1,
      it.label ?? it.key ?? '',
      Number(it.quantity ?? 0),
      centsToYuan(Number(it.amount ?? 0)),
    ]),
  );
}
</script>

<template>
  <Page :title="$t('menu.finance.salesRanking')">
    <div class="p-2">
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <a-select
          v-model:value="form.dimension"
          :options="dimensionOptions"
          style="width: 140px"
        />
        <a-date-picker
          v-model:value="form.fromDate"
          :placeholder="$t('page.trialBalance.fromDate')"
          value-format="YYYY-MM-DD"
          style="width: 150px"
        />
        <span>—</span>
        <a-date-picker
          v-model:value="form.toDate"
          :placeholder="$t('page.trialBalance.toDate')"
          value-format="YYYY-MM-DD"
          style="width: 150px"
        />
        <a-select v-model:value="form.limit" style="width: 90px">
          <a-select-option :value="5">Top 5</a-select-option>
          <a-select-option :value="10">Top 10</a-select-option>
          <a-select-option :value="20">Top 20</a-select-option>
        </a-select>
        <a-button type="primary" @click="load">
          {{ $t('ui.button.search') }}
        </a-button>
        <a-button :disabled="items.length === 0" @click="handleExport">
          {{ $t('page.salesRanking.export') }}
        </a-button>
      </div>

      <a-row :gutter="[16, 16]">
        <a-col :xs="24" :lg="14">
          <a-table
            :columns="columns"
            :data-source="items"
            :loading="loading"
            :pagination="false"
            row-key="key"
            size="small"
          >
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.key === 'rank'">
                <a-tag
                  v-if="index < 3"
                  color="gold"
                  class="mr-0"
                >
                  {{ index + 1 }}
                </a-tag>
                <span v-else>{{ index + 1 }}</span>
              </template>
              <template v-else-if="column.dataIndex === 'amount'">
                {{ centsToYuan(Number(record.amount ?? 0)) }}
              </template>
            </template>
            <template #emptyText>
              {{ $t('page.statement.empty') }}
            </template>
          </a-table>
        </a-col>
        <a-col :xs="24" :lg="10">
          <a-card :title="$t('page.salesRanking.chartTitle')" :bordered="false">
            <EchartsUI ref="chartRef" height="360px" />
          </a-card>
        </a-col>
      </a-row>
    </div>
  </Page>
</template>
