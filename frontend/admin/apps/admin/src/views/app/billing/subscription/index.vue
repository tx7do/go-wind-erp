<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import {
  apiClient,
  centsToYuanMonthly,
  fetchListPlans,
  fetchMySubscription,
  limitLabel,
  useChangePlan,
  useRenew,
} from '#/api';

interface PlanItem {
  code?: string;
  description?: string;
  id?: number;
  maxOrdersMonthly?: number;
  maxUsers?: number;
  name?: string;
  priceCents?: number;
}

defineOptions({ name: 'BillingSubscription' });

const loading = ref(true);
const acting = ref(false);

const usage = ref<Awaited<ReturnType<typeof fetchMySubscription>> | null>(null);
const plans = ref<PlanItem[]>([]);

const changePlanMutation = useChangePlan();
const renewMutation = useRenew();

async function load() {
  loading.value = true;
  try {
    const [u, p] = await Promise.all([
      fetchMySubscription(),
      fetchListPlans(),
    ]);
    usage.value = u;
    plans.value = (p?.items ?? []) as PlanItem[];
  } catch {
    notification.error({ message: $t('ui.notification.load_failed') });
  } finally {
    loading.value = false;
  }
}

onMounted(load);

function usagePercent(current: number, max?: number | null): number {
  if (!max || max <= 0) return 0;
  return Math.min(100, Math.round((current / max) * 100));
}

async function handleChangePlan(code?: string) {
  if (!code || acting.value) return;
  acting.value = true;
  try {
    await apiClient.billingService.ChangePlan({ planCode: code });
    notification.success({ message: $t('page.billing.planChanged') });
    await load();
  } catch {
    notification.error({ message: $t('page.billing.planChangeFailed') });
  } finally {
    acting.value = false;
  }
}

async function handleRenew() {
  if (acting.value) return;
  acting.value = true;
  try {
    await apiClient.billingService.Renew({});
    notification.success({ message: $t('page.billing.renewed') });
    await load();
  } catch {
    notification.error({ message: $t('page.billing.renewFailed') });
  } finally {
    acting.value = false;
  }
}

function isCurrentPlan(code?: string): boolean {
  return usage.value?.plan?.code != null && usage.value.plan.code === code;
}

function formatDate(v?: string | null): string {
  if (!v) return '—';
  return v.substring(0, 10);
}

void changePlanMutation;
void renewMutation;
</script>

<template>
  <Page :title="$t('page.billing.moduleName')">
    <div v-if="loading" class="flex justify-center py-12">
      <a-spin size="large" />
    </div>

    <div v-else-if="usage" class="flex flex-col gap-6 p-2">
      <!-- 当前订阅概览 -->
      <a-card>
        <template #title>
          {{ $t('page.billing.currentPlan') }}：
          {{ usage.plan?.name ?? 'FREE' }}
        </template>
        <template #extra>
          <a-alert
            v-if="usage.expired"
            type="error"
            show-icon
            :message="$t('page.billing.expiredBanner')"
            class="mr-4"
          />
        </template>

        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <div class="mb-1 text-sm text-gray-500">
              {{ $t('page.billing.userUsage') }}
            </div>
            <a-progress
              :percent="
                usagePercent(usage?.userCount ?? 0, usage?.plan?.maxUsers)
              "
              :format="
                () =>
                  `${usage?.userCount ?? 0} / ${limitLabel(usage?.plan?.maxUsers)}`
              "
            />
          </div>
          <div>
            <div class="mb-1 text-sm text-gray-500">
              {{ $t('page.billing.orderUsage') }}
            </div>
            <a-progress
              :percent="
                usagePercent(
                  usage?.orderCountMonth ?? 0,
                  usage?.plan?.maxOrdersMonthly,
                )
              "
              :format="
                () =>
                  `${usage?.orderCountMonth ?? 0} / ${limitLabel(usage?.plan?.maxOrdersMonthly)}`
              "
            />
          </div>
          <div>
            <div class="mb-1 text-sm text-gray-500">
              {{ $t('page.billing.expiry') }}
            </div>
            <div class="text-lg">
              {{
                usage?.subscription?.periodEnd
                  ? formatDate(usage.subscription?.periodEnd)
                  : $t('page.billing.permanent')
              }}
            </div>
            <a-button
              v-if="
                usage?.plan?.code && usage.plan.code !== 'FREE' && !usage.expired
              "
              class="mt-2"
              size="small"
              :loading="acting"
              @click="handleRenew"
            >
              {{ $t('page.billing.renew') }}
            </a-button>
          </div>
        </div>
      </a-card>

      <!-- 套餐选择（定价页） -->
      <div>
        <div class="mb-3 text-base font-medium">
          {{ $t('page.billing.choosePlan') }}
        </div>
        <div class="grid gap-4 md:grid-cols-3">
          <a-card
            v-for="p in plans"
            :key="p.code"
            :class="{ 'border-primary': isCurrentPlan(p.code) }"
          >
            <template #title>{{ p.name }}</template>
            <template #extra>
              <a-tag v-if="isCurrentPlan(p.code)" color="blue">
                {{ $t('page.billing.current') }}
              </a-tag>
            </template>
            <div class="mb-2 text-2xl font-bold">
              {{ centsToYuanMonthly(p.priceCents) }}
            </div>
            <p class="mb-2 text-sm text-gray-500">{{ p.description }}</p>
            <div class="mb-1 text-sm">
              {{ $t('page.billing.maxUsers') }}: {{ limitLabel(p.maxUsers) }}
            </div>
            <div class="mb-4 text-sm">
              {{ $t('page.billing.maxOrders') }}:
              {{ limitLabel(p.maxOrdersMonthly) }}
            </div>
            <a-button
              v-if="!isCurrentPlan(p.code)"
              type="primary"
              block
              :loading="acting"
              @click="handleChangePlan(p.code)"
            >
              {{
                (p.priceCents ?? 0) > 0
                  ? $t('page.billing.upgrade')
                  : $t('page.billing.downgrade')
              }}
            </a-button>
          </a-card>
        </div>
      </div>
    </div>
  </Page>
</template>
