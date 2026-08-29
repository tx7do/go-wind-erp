<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { exportRowsToExcel } from '#/utils/export-excel';

import {
  accountCategoryToName,
  centsToYuanLedger,
  fetchTrialBalance,
} from '#/api';

defineOptions({ name: 'TrialBalanceManagement' });

const loading = ref(true);
const items = ref<any[]>([]);
const totalDebit = ref(0);
const totalCredit = ref(0);
const fromDate = ref<string | undefined>(undefined);
const toDate = ref<string | undefined>(undefined);

async function load() {
  loading.value = true;
  try {
    const resp = await fetchTrialBalance({
      fromDate: fromDate.value ? new Date(`${fromDate.value}T00:00:00Z`) : undefined,
      toDate: toDate.value ? new Date(`${toDate.value}T23:59:59Z`) : undefined,
    } as any);
    items.value = resp?.items ?? [];
    totalDebit.value = resp?.totalDebit ?? 0;
    totalCredit.value = resp?.totalCredit ?? 0;
  } catch {
    notification.error({ message: $t('ui.notification.load_failed') });
  } finally {
    loading.value = false;
  }
}

onMounted(load);

function handleExport() {
  exportRowsToExcel(
    '科目余额表_' + new Date().toISOString().slice(0, 10),
    '科目余额表',
    ['科目编码', '科目名称', '类别', '借方累计', '贷方累计', '期末余额'],
    items.value.map((i: any) => [
      i.accountCode ?? '',
      i.accountName ?? '',
      accountCategoryToName(i.category),
      centsToYuanLedger(i.debitTotal),
      centsToYuanLedger(i.creditTotal),
      centsToYuanLedger(i.balance),
    ]),
  );
}

const columns = [
  { title: $t('page.trialBalance.accountCode'), dataIndex: 'accountCode' },
  { title: $t('page.trialBalance.accountName'), dataIndex: 'accountName' },
  {
    title: $t('page.trialBalance.category'),
    dataIndex: 'category',
    customRender: ({ text }: any) => accountCategoryToName(text),
  },
  { title: $t('page.trialBalance.debitTotal'), dataIndex: 'debitTotal', align: 'right' as const },
  { title: $t('page.trialBalance.creditTotal'), dataIndex: 'creditTotal', align: 'right' as const },
  { title: $t('page.trialBalance.balance'), dataIndex: 'balance', align: 'right' as const },
];
</script>

<template>
  <Page :title="$t('menu.finance.trialBalance')">
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
        <a-button :disabled="items.length === 0" @click="handleExport">
          {{ $t('page.salesRanking.export') }}
        </a-button>
        <div class="ml-auto flex items-center gap-2">
          <a-tag :color="totalDebit === totalCredit ? 'green' : 'red'">
            {{ $t('page.trialBalance.debitTotal') }}：
            {{ centsToYuanLedger(totalDebit) }}
          </a-tag>
          <a-tag :color="totalDebit === totalCredit ? 'green' : 'red'">
            {{ $t('page.trialBalance.creditTotal') }}：
            {{ centsToYuanLedger(totalCredit) }}
          </a-tag>
        </div>
      </div>

      <a-table
        :columns="columns"
        :data-source="items"
        :loading="loading"
        :pagination="false"
        row-key="accountCode"
      >
        <template #bodyCell="{ column, text }">
          <template v-if="column.dataIndex === 'debitTotal'">
            {{ centsToYuanLedger(text) }}
          </template>
          <template v-else-if="column.dataIndex === 'creditTotal'">
            {{ centsToYuanLedger(text) }}
          </template>
          <template v-else-if="column.dataIndex === 'balance'">
            {{ centsToYuanLedger(text) }}
          </template>
        </template>
      </a-table>
    </div>
  </Page>
</template>
