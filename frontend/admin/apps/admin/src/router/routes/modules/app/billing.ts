import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const billing: RouteRecordRaw[] = [
  {
    path: '/billing',
    name: 'Billing',
    component: BasicLayout,
    redirect: '/billing/subscription',
    meta: {
      order: 1006,
      icon: 'lucide:credit-card',
      title: $t('menu.billing.moduleName'),
      keepAlive: true,
    },
    children: [
      {
        path: 'subscription',
        name: 'BillingSubscription',
        meta: {
          order: 1,
          icon: 'lucide:badge-check',
          title: $t('menu.billing.subscription'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () =>
          import('#/views/app/billing/subscription/index.vue'),
      },
      {
        path: 'plans',
        name: 'BillingPlanManagement',
        meta: {
          order: 2,
          icon: 'lucide:layers',
          title: $t('menu.billing.plan'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/billing/plan/index.vue'),
      },
    ],
  },
];

export default billing;
