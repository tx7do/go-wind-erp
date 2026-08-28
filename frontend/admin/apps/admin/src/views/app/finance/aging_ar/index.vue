<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';

import { notification } from 'ant-design-vue';

import {
  centsToYuan,
  fetchAgingReportAr,
  type financeservicev1_AgingBucket,
} from '#/api';
import { $t } from '#/locales';

const loading = ref(true);
const buckets = ref<financeservicev1_AgingBucket[]>([]);

// 桶顺序固定（与后端 label 对应），缺失桶补零以保持表格完整。
const orderedBuckets = [
  'overdue',
  '0_7',
  '8_30',
  '31_90',
  'over_90',
  'no_due_date',
] as const;

const columns = [
  {
    title: $t('page.agingAr.bucket'),
    dataIndex: 'bucket',
    key: 'bucket',
    slots: { default: 'bucket' },
  },
  {
    title: $t('page.agingAr.count'),
    dataIndex: 'count',
    key: 'count',
  },
  {
    title: $t('page.agingAr.totalAmount'),
    dataIndex: 'totalAmount',
    key: 'totalAmount',
    slots: { default: 'amount' },
  },
];

async function load() {
  loading.value = true;
  try {
    const resp = await fetchAgingReportAr();
    const present = new Map<string, financeservicev1_AgingBucket>();
    for (const b of resp.buckets ?? []) {
      if (b?.bucket) {
        present.set(b.bucket, b);
      }
    }
    // 缺失桶补零——固定顺序输出。
    buckets.value = orderedBuckets.map(
      (label) =>
        present.get(label) ?? {
          bucket: label,
          count: 0,
          totalAmount: 0,
        },
    );
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
    <a-card :title="$t('page.agingAr.title')">
      <a-alert
        class="mb-3"
        type="warning"
        :message="$t('page.agingAr.disclaimer')"
        show-icon
      />
      <a-table
        :columns="columns"
        :data-source="buckets"
        :loading="loading"
        :pagination="false"
        row-key="bucket"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'bucket'">
            <a-tag :color="record.bucket === 'overdue' ? 'red' : 'default'">
              {{ $t('page.agingAr.bucketLabel.' + record.bucket) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'amount'">
            {{ centsToYuan(record.totalAmount) }}
          </template>
        </template>
      </a-table>
    </a-card>
  </Page>
</template>
