import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const warehouse: RouteRecordRaw[] = [
  {
    path: '/wms',
    name: 'Wms',
    component: BasicLayout,
    redirect: '/wms/warehouses',
    meta: {
      order: 1005,
      icon: 'lucide:warehouse',
      title: $t('menu.wms.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'warehouses',
        name: 'WarehouseManagement',
        meta: {
          order: 1,
          icon: 'lucide:warehouse',
          title: $t('menu.wms.warehouse'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/wms/warehouse/index.vue'),
      },

      {
        path: 'inventories',
        name: 'InventoryManagement',
        meta: {
          order: 2,
          icon: 'lucide:package',
          title: $t('menu.wms.inventory'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/wms/inventory/index.vue'),
      },

      {
        path: 'stock-movements',
        name: 'StockMovementManagement',
        meta: {
          order: 3,
          icon: 'lucide:arrow-right-left',
          title: $t('menu.wms.stockMovement'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/wms/stock_movement/index.vue'),
      },
    ],
  },
];

export default warehouse;
