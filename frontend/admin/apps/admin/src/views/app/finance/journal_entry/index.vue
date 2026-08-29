<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { centsToYuanLedger, fetchJournalEntries } from '#/api';

defineOptions({ name: 'JournalEntryManagement' });

const loading = ref(true);
const items = ref<any[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(20);
const fromDate = ref<string | undefined>(undefined);
const toDate = ref<string | undefined>(undefined);

async function load() {
  loading.value = true;
  try {
    const resp = await fetchJournalEntries({
      page: page.value,
      pageSize: pageSize.value,
      fromDate: fromDate.value
        ? new Date(`${fromDate.value}T00:00:00Z`)
        : undefined,
      toDate: toDate.value ? new Date(`${toDate.value}T23:59:59Z`) : undefined,
    } as any);
    items.value = resp?.items ?? [];
    total.value = Number(resp?.total ?? 0);
  } catch {
    notification.error({ message: $t('ui.notification.load_failed') });
  } finally {
    loading.value = false;
  }
}

onMounted(load);

const columns = [
  { title: $t('page.journal.entryNumber'), dataIndex: 'entryNumber', width: 170 },
  {
    title: $t('page.journal.entryDate'),
    dataIndex: 'entryDate',
    width: 130,
    customRender: ({ text }: any) => (text ? String(text).slice(0, 10) : '—'),
  },
  { title: $t('page.journal.summary'), dataIndex: 'summary' },
  { title: $t('page.journal.bizRef'), dataIndex: 'bizRef' },
  {
    title: $t('page.journal.amount'),
    key: 'amount',
    align: 'right' as const,
    width: 120,
  },
];

function entryAmount(record: any) {
  return centsToYuanLedger(
    (record.lines ?? []).reduce((s: number, l: any) => s + (l.debit ?? 0), 0),
  );
}
</script>

<template>
  <Page :title="$t('menu.finance.journal')">
    <div class="p-2">
      <div class="mb-2 flex flex-wrap items-center gap-2">
        <a-date-picker
          v-model:value="fromDate"
          :placeholder="$t('page.trialBalance.fromDate')"
          value-format="YYYY-MM-DD"
          style="width: 160px"
        />
        <span>—</span>
        <a-date-picker
          v-model:value="toDate"
          :placeholder="$t('page.trialBalance.toDate')"
          value-format="YYYY-MM-DD"
          style="width: 160px"
        />
        <a-button type="primary" @click="load">
          {{ $t('ui.button.search') }}
        </a-button>
      </div>

      <a-table
        :columns="columns"
        :data-source="items"
        :loading="loading"
        :pagination="{
          current: page,
          pageSize: pageSize,
          total: total,
          showSizeChanger: false,
          onChange: (p: number) => { page = p; load(); },
        }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'amount'">
            {{ entryAmount(record) }}
          </template>
        </template>

        <template #expandedRowRender="{ record }">
          <a-table
            :columns="[
              { title: $t('page.journal.account'), dataIndex: 'accountCode' },
              { title: $t('page.journal.lineSummary'), dataIndex: 'summary' },
              { title: $t('page.journal.debit'), dataIndex: 'debit', align: 'right' },
              { title: $t('page.journal.credit'), dataIndex: 'credit', align: 'right' },
            ]"
            :data-source="record.lines ?? []"
            :pagination="false"
            row-key="id"
            size="small"
          >
            <template #bodyCell="{ column: col, text }">
              <template v-if="col.dataIndex === 'debit'">
                {{ centsToYuanLedger(text) }}
              </template>
              <template v-else-if="col.dataIndex === 'credit'">
                {{ centsToYuanLedger(text) }}
              </template>
            </template>
          </a-table>
        </template>
      </a-table>
    </div>
  </Page>
</template>
