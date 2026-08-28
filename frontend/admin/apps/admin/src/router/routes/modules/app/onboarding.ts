import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const onboarding: RouteRecordRaw[] = [
  {
    path: '/onboarding',
    name: 'Onboarding',
    component: BasicLayout,
    redirect: '/onboarding/wizard',
    meta: {
      order: 999,
      icon: 'lucide:rocket',
      title: $t('page.onboarding.title'),
      hideInMenu: true,
    },
    children: [
      {
        path: 'wizard',
        name: 'OnboardingWizard',
        meta: {
          order: 1,
          icon: 'lucide:rocket',
          title: $t('page.onboarding.title'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/onboarding/index.vue'),
      },
    ],
  },
];

export default onboarding;
