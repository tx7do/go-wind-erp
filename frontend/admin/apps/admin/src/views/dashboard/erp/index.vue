<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { LucideBuilding, LucideInbox, LucideNotebookPen, LucidePencil } from '@vben/icons';

import { notification } from 'ant-design-vue';

import {
  fetchInventoryOverview,
  inventoryStatusToColor,
  inventoryStatusToName,
  type inventoryservicev1_InventoryOverview as InventoryOverview,
} from '#/api';
import { $t } from '#/locales';

const loading = ref(true);
const overview = ref<InventoryOverview>();

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

onMounted(() => {
  loadOverview();
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
