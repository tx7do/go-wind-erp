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
    ],
  },
];

export default finance;
