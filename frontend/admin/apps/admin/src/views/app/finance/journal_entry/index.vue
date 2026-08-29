<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { ref } from 'vue';

import { Page, type VbenFormProps } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { centsToYuanLedger, fetchJournalEntries } from '#/api';
import { $t } from '#/locales';
import { exportRowsToExcel } from '#/utils/export-excel';

defineOptions({ name: 'JournalEntryManagement' });

// 行数状态（导出按钮 disabled 用，避免模板里调 grid 实例方法）。
const rowCount = ref(0);

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'DatePicker',
      fieldName: 'fromDate',
      label: $t('page.trialBalance.fromDate'),
      componentProps: {
        valueFormat: 'YYYY-MM-DD',
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
      },
    },
    {
      component: 'DatePicker',
      fieldName: 'toDate',
      label: $t('page.trialBalance.toDate'),
      componentProps: {
        valueFormat: 'YYYY-MM-DD',
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<any> = {
  toolbarConfig: { custom: true, refresh: true, zoom: true },
  height: 'auto',
  exportConfig: {},
  pagerConfig: {},
  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        const resp = await fetchJournalEntries({
          page: page.currentPage,
          pageSize: page.pageSize,
          fromDate: formValues?.fromDate
            ? new Date(`${formValues.fromDate}T00:00:00Z`)
            : undefined,
          toDate: formValues?.toDate
            ? new Date(`${formValues.toDate}T23:59:59Z`)
            : undefined,
        } as any);
        const items = resp?.items ?? [];
        rowCount.value = Number(resp?.total ?? 0);
        return { items, total: Number(resp?.total ?? 0) };
      },
    },
  },
  columns: [
    { type: 'expand', width: 50, slots: { content: 'expandContent' } },
    {
      title: $t('page.journal.entryNumber'),
      field: 'entryNumber',
      width: 170,
    },
    {
      title: $t('page.journal.entryDate'),
      field: 'entryDate',
      formatter: ({ cellValue }) =>
        cellValue ? String(cellValue).slice(0, 10) : '—',
      width: 120,
    },
    { title: $t('page.journal.summary'), field: 'summary' },
    { title: $t('page.journal.bizRef'), field: 'bizRef' },
    {
      title: $t('page.journal.amount'),
      field: 'amount',
      align: 'right',
      formatter: ({ row }) =>
        centsToYuanLedger(
          (row.lines ?? []).reduce((s: number, l: any) => s + (l.debit ?? 0), 0),
        ),
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

function handleExport() {
  const tableData = gridApi.grid?.getTableData().fullData ?? [];
  exportRowsToExcel(
    '凭证流水_' + new Date().toISOString().slice(0, 10),
    '凭证流水',
    ['凭证号', '日期', '摘要', '业务来源', '金额'],
    tableData.map((e: any) => [
      e.entryNumber ?? '',
      String(e.entryDate ?? '').slice(0, 10),
      e.summary ?? '',
      e.bizRef ?? '',
      centsToYuanLedger(
        (e.lines ?? []).reduce((s: number, l: any) => s + (l.debit ?? 0), 0),
      ),
    ]),
  );
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.finance.journal')">
      <template #toolbar-tools>
        <a-button
          class="mr-2"
          :disabled="rowCount === 0"
          @click="handleExport"
        >
          {{ $t('page.salesRanking.export') }}
        </a-button>
      </template>
      <!-- 分录行放在展开行（与原 a-table expandedRowRender 等价） -->
      <template #expandContent="{ row }">
        <a-table
          :columns="[
            { title: $t('page.journal.account'), dataIndex: 'accountCode' },
            { title: $t('page.journal.lineSummary'), dataIndex: 'summary' },
            { title: $t('page.journal.debit'), dataIndex: 'debit', align: 'right' },
            { title: $t('page.journal.credit'), dataIndex: 'credit', align: 'right' },
          ]"
          :data-source="row.lines ?? []"
          :pagination="false"
          row-key="id"
          size="small"
        >
          <template #bodyCell="{ column, text }">
            <template v-if="column.dataIndex === 'debit'">
              {{ centsToYuanLedger(text) }}
            </template>
            <template v-else-if="column.dataIndex === 'credit'">
              {{ centsToYuanLedger(text) }}
            </template>
          </template>
        </a-table>
      </template>
    </Grid>
  </Page>
</template>
