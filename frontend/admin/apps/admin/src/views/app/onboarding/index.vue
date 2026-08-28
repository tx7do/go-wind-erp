<script lang="ts" setup>
import { useRouter } from 'vue-router';

import { Page } from '@vben/common-ui';

import { $t } from '@vben/locales';

defineOptions({ name: 'OnboardingWizard' });

const router = useRouter();

interface Step {
  key: string;
  route: string;
}

// 后端 SelfRegisterTenant 已自动完成：租户角色、SUPPLIER/CUSTOMER 虚拟位置、
// 默认仓库 MAIN。向导引导用户完成剩余三步：商品 → 往来单位 → 首单。
const steps: Step[] = [
  { key: 'product', route: '/wms/products' },
  { key: 'supplier', route: '/procurement/suppliers' },
  { key: 'customer', route: '/sales/customers' },
  { key: 'po', route: '/procurement/purchase-orders' },
  { key: 'so', route: '/sales/sales-orders' },
];

const stepMeta: Record<string, { icon: string; desc: string }> = {
  product: {
    icon: '📦',
    desc: 'page.onboarding.stepProductDesc',
  },
  supplier: {
    icon: '🚚',
    desc: 'page.onboarding.stepSupplierDesc',
  },
  customer: {
    icon: '🏪',
    desc: 'page.onboarding.stepCustomerDesc',
  },
  po: {
    icon: '🧾',
    desc: 'page.onboarding.stepPoDesc',
  },
  so: {
    icon: '💰',
    desc: 'page.onboarding.stepSoDesc',
  },
};

function goTo(route: string) {
  router.push(route);
}
</script>

<template>
  <Page :title="$t('page.onboarding.title')" :description="$t('page.onboarding.desc')">
    <div class="p-4">
      <!-- 已自动完成 -->
      <div class="mb-6 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-900 dark:bg-green-950">
        <div class="mb-2 font-medium text-green-800 dark:text-green-300">
          ✅ {{ $t('page.onboarding.autoTitle') }}
        </div>
        <div class="text-sm text-green-700 dark:text-green-400">
          {{ $t('page.onboarding.autoDesc') }}
        </div>
      </div>

      <!-- 引导步骤 -->
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="(step, i) in steps"
          :key="step.key"
          class="cursor-pointer rounded-lg border p-4 transition-shadow hover:shadow-md"
          @click="goTo(step.route)"
        >
          <div class="mb-2 flex items-center gap-2">
            <span class="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 font-semibold text-primary">
              {{ i + 1 }}
            </span>
            <span class="text-xl">{{ stepMeta[step.key]?.icon }}</span>
          </div>
          <div class="mb-1 font-medium">
            {{ $t(`page.onboarding.step${step.key.charAt(0).toUpperCase()}${step.key.slice(1)}`) }}
          </div>
          <div class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t(stepMeta[step.key]?.desc ?? '') }}
          </div>
        </div>
      </div>
    </div>
  </Page>
</template>
