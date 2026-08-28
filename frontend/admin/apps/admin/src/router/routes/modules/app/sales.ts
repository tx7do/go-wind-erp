import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const sales: RouteRecordRaw[] = [
  {
    path: '/sales',
    name: 'Sales',
    component: BasicLayout,
    redirect: '/sales/sales-orders',
    meta: {
      order: 1010,
      icon: 'lucide:store',
      title: $t('menu.sales.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:sales'],
    },
    children: [
      {
        path: 'sales-orders',
        name: 'SalesOrderManagement',
        meta: {
          order: 1,
          icon: 'lucide:notebook-pen',
          title: $t('menu.sales.salesOrder'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:sales'],
        },
        component: () => import('#/views/app/sales/sales_order/index.vue'),
      },
      {
        path: 'customers',
        name: 'CustomerManagement',
        meta: {
          order: 2,
          icon: 'lucide:building',
          title: $t('menu.sales.customer'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:sales'],
        },
        component: () => import('#/views/app/sales/customer/index.vue'),
      },
    ],
   },
];

export default sales;
