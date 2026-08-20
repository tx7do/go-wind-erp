import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const approval: RouteRecordRaw[] = [
  {
    path: '/approval',
    name: 'Approval',
    component: BasicLayout,
    redirect: '/approval/requests',
    meta: {
      order: 1010,
      icon: 'lucide:clipboard-check',
      title: $t('menu.approval.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'requests',
        name: 'ApprovalRequestManagement',
        meta: {
          order: 1,
          icon: 'lucide:clipboard-check',
          title: $t('menu.approval.request'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/approval/index.vue'),
      },
    ],
  },
];

export default approval;
