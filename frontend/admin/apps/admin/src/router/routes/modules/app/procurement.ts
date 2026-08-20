import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const procurement: RouteRecordRaw[] = [
  {
    path: '/procurement',
    name: 'Procurement',
    component: BasicLayout,
    redirect: '/procurement/purchase-orders',
    meta: {
      order: 1008,
      icon: 'lucide:truck',
      title: $t('menu.procurement.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'purchase-orders',
        name: 'PurchaseOrderManagement',
        meta: {
          order: 1,
          icon: 'lucide:notebook-pen',
          title: $t('menu.procurement.purchaseOrder'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/procurement/purchase_order/index.vue'),
      },
      {
        path: 'suppliers',
        name: 'SupplierManagement',
        meta: {
          order: 2,
          icon: 'lucide:building',
          title: $t('menu.procurement.supplier'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/procurement/supplier/index.vue'),
      },
    ],
  },
];

export default procurement;
