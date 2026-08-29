<script lang="ts" setup>
import { onMounted, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { exportRowsToExcel } from '#/utils/export-excel';

import {
  apiClient,
  centsToYuan,
  fetchListCustomers,
  fetchListSuppliers,
  PaginationQuery,
} from '#/api';
import { $t as $tx } from '#/locales';
import { escapeHtml, printHtml } from '#/utils/print';

defineOptions({ name: 'PartnerStatementManagement' });

const loading = ref(false);
const form = reactive({
  partnerType: 'CUSTOMER',
  partnerCode: '' as string,
  fromDate: undefined as string | undefined,
  toDate: undefined as string | undefined,
});
const partnerOptions = ref<{ label: string; value: string }[]>([]);
const statement = ref<any>(null);

const partnerTypeOptions = [
  { value: 'CUSTOMER', label: $t('page.statement.customer') },
  { value: 'SUPPLIER', label: $t('page.statement.supplier') },
];

async function loadPartners() {
  try {
    const fetcher =
      form.partnerType === 'CUSTOMER' ? fetchListCustomers : fetchListSuppliers;
    const resp = await fetcher(
      new PaginationQuery({ paging: { page: 1, pageSize: 500 } }),
    );
    partnerOptions.value = (resp?.items ?? [])
      .filter((p: any) => p.enable !== false)
      .map((p: any) => ({ label: `${p.code} — ${p.name}`, value: p.code }));
  } catch {
    // 下拉失败不阻塞页面
  }
}

function onTypeChange() {
  form.partnerCode = '';
  loadPartners();
}

async function load() {
  if (!form.partnerCode) {
    notification.warning({ message: $t('page.statement.partnerRequired') });
    return;
  }
  loading.value = true;
  try {
    statement.value = await apiClient.financeReportService.GetPartnerStatement({
      partnerType: form.partnerType,
      partnerCode: form.partnerCode,
      fromDate: form.fromDate
        ? new Date(`${form.fromDate}T00:00:00Z`)
        : undefined,
      toDate: form.toDate ? new Date(`${form.toDate}T23:59:59Z`) : undefined,
    } as any);
  } catch {
    notification.error({ message: $t('ui.notification.load_failed') });
  } finally {
    loading.value = false;
  }
}

onMounted(loadPartners);

function handleExport() {
  const st = statement.value;
  if (!st) return;
  exportRowsToExcel(
    `对账单_${form.partnerCode}_${new Date().toISOString().slice(0, 10)}`,
    '对账单',
    ['日期', '类型', '单据', '摘要', '发生额', '核销额'],
    (st.rows ?? []).map((r: any) => [
      String(r.date ?? '').slice(0, 10),
      r.docType ?? '',
      r.docRef ?? '',
      r.summary ?? '',
      fmt(r.debit),
      fmt(r.credit),
    ]),
  );
}

const columns = [
  {
    title: $t('page.statement.date'),
    dataIndex: 'date',
    width: 120,
    customRender: ({ text }: any) => (text ? String(text).slice(0, 10) : '—'),
  },
  { title: $t('page.statement.docType'), dataIndex: 'docType', width: 100 },
  { title: $t('page.statement.docRef'), dataIndex: 'docRef' },
  { title: $t('page.statement.summary'), dataIndex: 'summary' },
  {
    title: $t('page.statement.debit'),
    dataIndex: 'debit',
    align: 'right' as const,
    width: 120,
  },
  {
    title: $t('page.statement.credit'),
    dataIndex: 'credit',
    align: 'right' as const,
    width: 120,
  },
];

function fmt(v: any) {
  return centsToYuan(Number(v ?? 0));
}

/** 打印对账单（复用 iframe 打印底座）。 */
function handlePrint() {
  const st = statement.value;
  if (!st) return;
  const partnerLabel =
    partnerOptions.value.find((p) => p.value === form.partnerCode)?.label ??
    form.partnerCode;
  const typeLabel =
    form.partnerType === 'CUSTOMER'
      ? $t('page.statement.customer')
      : $t('page.statement.supplier');
  const rows = (st.rows ?? [])
    .map(
      (r: any) =>
        `<tr><td>${escapeHtml(String(r.date ?? '').slice(0, 10))}</td><td>${escapeHtml(
          r.docType ?? '',
        )}</td><td>${escapeHtml(r.docRef ?? '')}</td><td>${escapeHtml(
          r.summary ?? '',
        )}</td><td class="num">${escapeHtml(fmt(r.debit))}</td><td class="num">${escapeHtml(
          fmt(r.credit),
        )}</td></tr>`,
    )
    .join('');
  const html = `
<div class="doc">
  <h1 class="doc-title">${escapeHtml($t('page.statement.title'))}</h1>
  <div class="doc-sub">${escapeHtml(typeLabel)}：${escapeHtml(partnerLabel)}　·　${escapeHtml(
    $t('page.doc.printTime'),
  )}：${new Date().toLocaleString('zh-CN')}</div>
  <table class="doc-items">
    <thead><tr>
      <th>${escapeHtml($t('page.statement.date'))}</th>
      <th>${escapeHtml($t('page.statement.docType'))}</th>
      <th>${escapeHtml($t('page.statement.docRef'))}</th>
      <th>${escapeHtml($t('page.statement.summary'))}</th>
      <th>${escapeHtml($t('page.statement.debit'))}</th>
      <th>${escapeHtml($t('page.statement.credit'))}</th>
    </tr></thead>
    <tbody>${rows}</tbody>
  </table>
  <div class="doc-total">
    <span>${escapeHtml($t('page.statement.totalDebit'))}：${escapeHtml(fmt(st.totalDebit))}</span>
    <span>${escapeHtml($t('page.statement.totalCredit'))}：${escapeHtml(fmt(st.totalCredit))}</span>
    <span class="amount">${escapeHtml($t('page.statement.balance'))}：${escapeHtml(fmt(st.balance))}</span>
  </div>
  <table class="doc-sign"><tr>
    <td>${escapeHtml($tx('page.doc.sign.creator'))}：</td>
    <td>${escapeHtml($t('page.statement.partnerConfirm'))}：</td>
  </tr></table>
</div>`;
  printHtml($t('page.statement.title'), html);
}
</script>

<template>
  <Page :title="$t('menu.finance.statement')">
    <div class="p-2">
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <a-select
          v-model:value="form.partnerType"
          :options="partnerTypeOptions"
          style="width: 120px"
          @change="onTypeChange"
        />
        <a-select
          v-model:value="form.partnerCode"
          :options="partnerOptions"
          :placeholder="$t('page.statement.partnerPlaceholder')"
          show-search
          option-filter-prop="label"
          style="width: 260px"
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
        <a-button type="primary" @click="load">
          {{ $t('ui.button.search') }}
        </a-button>
        <a-button
          v-if="statement"
          :disabled="(statement.rows ?? []).length === 0"
          @click="handleExport"
        >
          {{ $t('page.salesRanking.export') }}
        </a-button>
        <a-button v-if="statement" @click="handlePrint">
          {{ $t('page.purchaseOrder.button.print') }}
        </a-button>
      </div>

      <template v-if="statement">
        <div class="mb-2 flex gap-2">
          <a-tag color="blue">
            {{ $t('page.statement.totalDebit') }}：{{ fmt(statement.totalDebit) }}
          </a-tag>
          <a-tag color="green">
            {{ $t('page.statement.totalCredit') }}：{{
              fmt(statement.totalCredit)
            }}
          </a-tag>
          <a-tag color="orange">
            {{ $t('page.statement.balance') }}：{{ fmt(statement.balance) }}
          </a-tag>
        </div>

        <a-table
          :columns="columns"
          :data-source="statement.rows ?? []"
          :loading="loading"
          :pagination="false"
          row-key="docRef"
          size="small"
        >
          <template #bodyCell="{ column, text }">
            <template v-if="column.dataIndex === 'debit'">
              {{ fmt(text) }}
            </template>
            <template v-else-if="column.dataIndex === 'credit'">
              {{ fmt(text) }}
            </template>
          </template>
          <template #emptyText>
            {{ $t('page.statement.empty') }}
          </template>
        </a-table>
      </template>
      <a-empty v-else :description="$t('page.statement.empty')" />
    </div>
  </Page>
</template>
