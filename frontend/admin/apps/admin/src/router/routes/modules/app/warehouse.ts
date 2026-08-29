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
      authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:warehouse_keeper'],
    },
    children: [
      {
        path: 'warehouses',
        name: 'WarehouseManagement',
        meta: {
          order: 1,
          icon: 'lucide:warehouse',
          title: $t('menu.wms.warehouse'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:warehouse_keeper'],
        },
        component: () => import('#/views/app/wms/warehouse/index.vue'),
      },

      {
        path: 'stock-quants',
        name: 'StockQuantManagement',
        meta: {
          order: 2,
          icon: 'lucide:package',
          title: $t('menu.wms.stockQuant'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:warehouse_keeper'],
        },
        component: () => import('#/views/app/wms/inventory/index.vue'),
      },

      {
        path: 'products',
        name: 'ProductManagement',
        meta: {
          order: 0,
          icon: 'lucide:package',
          title: $t('menu.wms.product'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:warehouse_keeper'],
        },
        component: () => import('#/views/app/wms/product/index.vue'),
      },

      {
        path: 'stock-pickings',
        name: 'StockPickingManagement',
        meta: {
          order: 3,
          icon: 'lucide:arrow-right-left',
          title: $t('menu.wms.stockPicking'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:warehouse_keeper'],
        },
        component: () => import('#/views/app/wms/stock_movement/index.vue'),
      },

      {
        path: 'stock-lots',
        name: 'StockLotManagement',
        meta: {
          order: 4,
          icon: 'lucide:calendar-clock',
          title: $t('menu.wms.stockLot'),
          authority: ['sys:platform_admin', 'sys:tenant_manager', 'sys:warehouse_keeper'],
        },
        component: () => import('#/views/app/wms/stock_lot/index.vue'),
      },
    ],
  },
];

export default warehouse;
