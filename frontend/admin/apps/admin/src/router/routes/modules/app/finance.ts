import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const finance: RouteRecordRaw[] = [
  {
    path: '/finance',
    name: 'Finance',
    component: BasicLayout,
    redirect: '/finance/payables',
    meta: {
      order: 1009,
      icon: 'lucide:circle-dollar-sign',
      title: $t('menu.finance.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
    },
    children: [
      {
        path: 'aging',
        name: 'PayableAgingReport',
        meta: {
          order: 0,
          icon: 'lucide:alarm-clock',
          title: $t('menu.finance.aging'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/aging/index.vue'),
      },

      {
        path: 'trial-balance',
        name: 'TrialBalanceReport',
        meta: {
          order: 1,
          icon: 'lucide:scale',
          title: $t('menu.finance.trialBalance'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/trial_balance/index.vue'),
      },

      {
        path: 'sales-ranking',
        name: 'SalesRankingReport',
        meta: {
          order: 4,
          icon: 'lucide:bar-chart-3',
          title: $t('menu.finance.salesRanking'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/sales_ranking/index.vue'),
      },

      {
        path: 'partner-statement',
        name: 'PartnerStatement',
        meta: {
          order: 3,
          icon: 'lucide:file-text',
          title: $t('menu.finance.statement'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/statement/index.vue'),
      },

      {
        path: 'journal-entries',
        name: 'JournalEntryManagement',
        meta: {
          order: 2,
          icon: 'lucide:book-open',
          title: $t('menu.finance.journal'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/journal_entry/index.vue'),
      },
      {
        path: 'aging-ar',
        name: 'ReceivableAgingReport',
        meta: {
          order: 0,
          icon: 'lucide:alarm-clock',
          title: $t('menu.finance.agingAr'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/aging_ar/index.vue'),
      },
      {
        path: 'profit',
        name: 'ProfitReport',
        meta: {
          order: 0,
          icon: 'lucide:chart-line',
          title: $t('menu.finance.profit'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/profit/index.vue'),
      },
      {
        path: 'payables',
        name: 'PayableManagement',
        meta: {
          order: 1,
          icon: 'lucide:notebook-pen',
          title: $t('menu.finance.payable'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/payable/index.vue'),
      },
      {
        path: 'receivables',
        name: 'ReceivableManagement',
        meta: {
          order: 1,
          icon: 'lucide:notebook-pen',
          title: $t('menu.finance.receivable'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/receivable/index.vue'),
      },
      {
        path: 'payments',
        name: 'PaymentManagement',
        meta: {
          order: 2,
          icon: 'lucide:download',
          title: $t('menu.finance.payment'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/payment/index.vue'),
      },
      {
        path: 'receipts',
        name: 'ReceiptManagement',
        meta: {
          order: 2,
          icon: 'lucide:download',
          title: $t('menu.finance.receipt'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:finance'],
        },
        component: () => import('#/views/app/finance/receipt/index.vue'),
      },
    ],
  },
];

export default finance;
