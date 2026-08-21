<script lang="ts" setup>
import { onMounted, ref, shallowRef, watch } from 'vue';

import type { EchartsUIType } from '@vben/plugins/echarts';

import { LucideBuilding, LucideInbox, LucideNotebookPen, LucidePencil } from '@vben/icons';
import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

import { notification } from 'ant-design-vue';

import {
  fetchInventoryOverview,
  fetchMovementTrend,
  inventoryStatusToColor,
  inventoryStatusToName,
  type inventoryservicev1_InventoryOverview as InventoryOverview,
  type inventoryservicev1_MovementTrendResponse as MovementTrendResponse,
} from '#/api';
import { $t } from '#/locales';

const chartRef = shallowRef<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const loading = ref(true);
const overview = ref<InventoryOverview>();
const trend = ref<MovementTrendResponse>();

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

const lowStockColumns = [
  { title: $t('page.inventory.warehouseCode'), dataIndex: 'warehouseCode' },
  { title: $t('page.inventory.skuCode'), dataIndex: 'skuCode' },
  { title: $t('page.inventory.quantity'), dataIndex: 'quantity' },
  {
    title: $t('page.inventory.status'),
    dataIndex: 'status',
    key: 'status',
  },
];

async function loadOverview() {
  loading.value = true;
  try {
    overview.value = await fetchInventoryOverview({});
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  } finally {
    loading.value = false;
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
          :row-key="(record: any) => `${record.warehouseCode}-${record.skuCode}`"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag :color="inventoryStatusToColor(record.status)">
                {{ inventoryStatusToName(record.status) }}
              </a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'quantity'">
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
