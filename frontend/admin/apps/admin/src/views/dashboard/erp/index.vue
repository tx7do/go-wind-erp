<script lang="ts" setup>
import { computed, onMounted, ref, shallowRef, watch } from 'vue';

import type { EchartsUIType } from '@vben/plugins/echarts';

import {
  LucideBuilding,
  LucideCoins,
  LucideInbox,
  LucideNotebookPen,
  LucidePencil,
  LucideTrendingDown,
  LucideTrendingUp,
  LucideWallet,
} from '@vben/icons';
import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

import { notification } from 'ant-design-vue';

import {
  centsToYuan,
  fetchFinanceSummary,
  fetchListStockLots,
  fetchMovementTrend,
  fetchStockQuantOverview,
  formatLotExpiry,
  PaginationQuery,
  type financeservicev1_FinanceSummaryResponse as FinanceSummary,
  type inventoryservicev1_MovementTrendResponse as MovementTrendResponse,
  type inventoryservicev1_StockQuantOverview as StockQuantOverview,
} from '#/api';
import { $t } from '#/locales';

const chartRef = shallowRef<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const loading = ref(true);
const overview = ref<StockQuantOverview>();
const trend = ref<MovementTrendResponse>();
const summary = ref<FinanceSummary>();

const metricCards = [
  {
    key: 'warehouseCount',
    icon: LucideBuilding,
    color: '#1677ff',
  },
  {
    key: 'skuCount',
    icon: LucideInbox,
    color: '#13c2c2',
  },
  {
    key: 'totalQuantity',
    icon: LucideNotebookPen,
    color: '#2f54eb',
  },
  {
    key: 'movementCount',
    icon: LucidePencil,
    color: '#fa541c',
  },
] as const;

const financeCards = computed(() => [
  {
    key: 'revenueMonth',
    icon: LucideTrendingUp,
    color: '#16a34a',
    value: centsToYuan(summary.value?.revenueMonth ?? 0),
  },
  {
    key: 'cogsMonth',
    icon: LucideTrendingDown,
    color: '#f97316',
    value: centsToYuan(summary.value?.cogsMonth ?? 0),
  },
  {
    key: 'profitMonth',
    icon: LucideCoins,
    color: '#1677ff',
    value: centsToYuan(summary.value?.profitMonth ?? 0),
  },
  {
    key: 'arApBalance',
    icon: LucideWallet,
    color: '#722ed1',
    value:
      centsToYuan(summary.value?.arBalance ?? 0) +
      ' / ' +
      centsToYuan(summary.value?.apBalance ?? 0),
  },
]);

const expiringLots = ref<any[]>([]);
const expiringColumns = [
  { title: $t('page.stockLot.name'), dataIndex: 'name' },
  { title: $t('page.stockLot.skuCode'), dataIndex: 'skuCode' },
  {
    title: $t('page.stockLot.expiryDate'),
    dataIndex: 'expiryDate',
    customRender: ({ text }: any) => formatLotExpiry(text),
  },
  { title: $t('page.stockLot.remainingQuantity'), dataIndex: 'remainingQuantity' },
];

const lowStockColumns = [
  { title: $t('page.stockQuant.locationId'), dataIndex: 'locationId' },
  { title: $t('page.stockQuant.productCode'), dataIndex: 'productCode' },
  { title: $t('page.stockQuant.quantity'), dataIndex: 'quantity' },
];

async function loadOverview() {
  loading.value = true;
  try {
    overview.value = await fetchStockQuantOverview({});
  } catch {
    notification.error({
          message: $t('ui.notification.operation_failed'),
        });
  } finally {
    loading.value = false;
  }
}

async function loadFinance() {
  try {
    summary.value = await fetchFinanceSummary();
  } catch {
    // 汇总失败不阻塞看板
  }
  try {
    const lots = await fetchListStockLots(
      new PaginationQuery({
        paging: { page: 1, pageSize: 50 },
        formValues: { lotStatus: 'LOT_EXPIRING' },
      }),
    );
    expiringLots.value = (lots?.items ?? []).filter(
      (l: any) => (l.remainingQuantity ?? 0) > 0,
    );
  } catch {
    // 批次拉取失败不阻塞看板
  }
}

async function loadTrend() {
  try {
    trend.value = await fetchMovementTrend();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}

watch(trend, (val) => {
  if (!val?.points?.length) return;
  const pts = val.points.filter((p) => p.date !== undefined && p.count !== undefined);
  if (!pts.length) return;
  renderEcharts({
    grid: { top: 30, right: 20, bottom: 30, left: 40 },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: pts.map((p) => p.date as string),
    },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{ type: 'line', smooth: true, areaStyle: {}, data: pts.map((p) => p.count as number) }],
  });
});

onMounted(() => {
  loadOverview();
  loadTrend();
  loadFinance();
});
</script>

<template>
  <div class="p-5">
    <a-spin :spinning="loading">
      <a-row :gutter="[16, 16]">
        <a-col
          v-for="card in metricCards"
          :key="card.key"
          :xs="24"
          :sm="12"
          :lg="6"
        >
          <a-card>
            <div class="flex items-center gap-3">
              <div
                class="flex h-11 w-11 items-center justify-center rounded-lg"
                :style="{ backgroundColor: `${card.color}1a` }"
              >
                <component
                  :is="card.icon"
                  class="h-5 w-5"
                  :style="{ color: card.color }"
                />
              </div>
              <div>
                <div class="text-xs text-gray-500">
                  {{ $t(`page.dashboard.${card.key}`) }}
                </div>
                <div class="text-2xl font-semibold">
                  {{ overview?.[card.key] ?? '-' }}
                </div>
              </div>
            </div>
          </a-card>
        </a-col>
      </a-row>

      <a-row class="mt-4" :gutter="[16, 16]">
        <a-col
          v-for="card in financeCards"
          :key="card.key"
          :xs="24"
          :sm="12"
          :lg="6"
        >
          <a-card>
            <div class="flex items-center gap-3">
              <div
                class="flex h-11 w-11 items-center justify-center rounded-lg"
                :style="{ backgroundColor: `${card.color}1a` }"
              >
                <component
                  :is="card.icon"
                  class="h-5 w-5"
                  :style="{ color: card.color }"
                />
              </div>
              <div>
                <div class="text-xs text-gray-500">
                  {{ $t(`page.dashboard.${card.key}`) }}
                </div>
                <div class="text-2xl font-semibold">{{ card.value }}</div>
              </div>
            </div>
          </a-card>
        </a-col>
      </a-row>

      <a-card
        v-if="expiringLots.length > 0"
        class="mt-4"
        :title="$t('page.dashboard.expiringLotsTitle')"
        :body-style="{ padding: 0 }"
      >
        <template #extra>
          <span class="text-sm text-orange-500">
            {{ $t('page.dashboard.expiringLotsCount', { count: expiringLots.length }) }}
          </span>
        </template>
        <a-table
          :columns="expiringColumns"
          :data-source="expiringLots"
          :pagination="false"
          row-key="id"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'expiryDate'">
              <a-tag color="orange">{{ formatLotExpiry(record.expiryDate) }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'remainingQuantity'">
              <span class="font-semibold">{{ record.remainingQuantity }}</span>
            </template>
          </template>
        </a-table>
      </a-card>

      <a-card class="mt-4" :title="$t('page.dashboard.movementTrendTitle')">
        <EchartsUI ref="chartRef" height="320px" />
      </a-card>

      <a-card
        class="mt-4"
        :title="$t('page.dashboard.lowStockTitle')"
        :body-style="{ padding: 0 }"
      >
        <a-table
          :columns="lowStockColumns"
          :data-source="overview?.lowStockItems ?? []"
          :pagination="false"
          :row-key="(record: any) => `${record.locationId}-${record.productCode}`"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'quantity'">
              <span class="font-semibold text-red-500">
                {{ record.quantity }}
              </span>
            </template>
          </template>
          <template #emptyText>
            {{ $t('page.dashboard.lowStockEmpty') }}
          </template>
        </a-table>
      </a-card>
    </a-spin>
  </div>
</template>
