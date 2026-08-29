<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import {
  apiClient,
  fetchListApprovalFlows,
  fetchListRoles,
  flowBizTypeToName,
  PaginationQuery,
  type approvalservicev1_ApprovalFlow as ApprovalFlow,
} from '#/api';

defineOptions({ name: 'ApprovalFlowManagement' });

const loading = ref(true);
const flows = ref<ApprovalFlow[]>([]);
const roleOptions = ref<{ label: string; value: string }[]>([]);

const modalOpen = ref(false);
const editingId = ref<number | null>(null);
const form = ref({ bizType: '', name: '' });
// 级编辑行：seq 由数组顺序隐含（提交时 1..N 重排）
const stepRows = ref<{ name: string; roleCode: string }[]>([]);
const saving = ref(false);

async function load() {
  loading.value = true;
  try {
    const resp = await fetchListApprovalFlows(
      new PaginationQuery({ paging: { page: 1, pageSize: 100 } }),
    );
    flows.value = (resp?.items ?? []) as ApprovalFlow[];
  } catch {
    notification.error({ message: $t('ui.notification.load_failed') });
  } finally {
    loading.value = false;
  }
}

async function loadRoles() {
  try {
    const resp = await fetchListRoles(
      new PaginationQuery({ paging: { page: 1, pageSize: 200 } }),
    );
    roleOptions.value = (resp?.items ?? [])
      .filter((r: any) => r.code)
      .map((r: any) => ({
        label: `${r.code} — ${r.name}`,
        value: r.code,
      }));
  } catch {
    // 角色拉取失败时下拉为空，保存仍可用（手输不可行则重进页面）
  }
}

onMounted(() => {
  load();
  loadRoles();
});

const bizTypeOptions = [
  { value: 'PURCHASE_ORDER', label: $t('page.approvalFlow.bizType.PURCHASE_ORDER') },
  { value: 'SALES_ORDER', label: $t('page.approvalFlow.bizType.SALES_ORDER') },
  { value: 'PAYMENT', label: $t('page.approvalFlow.bizType.PAYMENT') },
  { value: 'RECEIPT', label: $t('page.approvalFlow.bizType.RECEIPT') },
];

const columns = [
  { title: $t('page.approvalFlow.bizTypeLabel'), dataIndex: 'bizType' },
  { title: $t('page.approvalFlow.name'), dataIndex: 'name' },
  { title: $t('page.approvalFlow.stepsCount'), key: 'stepsCount', width: 90 },
  { title: $t('page.approvalFlow.stepsOverview'), key: 'steps' },
  { title: $t('ui.table.createdAt'), dataIndex: 'createdAt', width: 150 },
  { title: $t('ui.table.action'), key: 'op', width: 110 },
];

function stepsText(record: any) {
  return (record.steps ?? [])
    .map((st: any, i: number) => `${i + 1}. ${st.name} [${st.roleCode}]`)
    .join(' → ');
}

function handleCreate() {
  editingId.value = null;
  form.value = { bizType: 'PURCHASE_ORDER', name: '' };
  stepRows.value = [{ name: '', roleCode: '' }];
  modalOpen.value = true;
}

function handleEdit(record: any) {
  editingId.value = record.id ?? null;
  form.value = { bizType: record.bizType ?? '', name: record.name ?? '' };
  stepRows.value = (record.steps ?? []).map((st: any) => ({
    name: st.name ?? '',
    roleCode: st.roleCode ?? '',
  }));
  modalOpen.value = true;
}

async function handleDelete(record: any) {
  try {
    await apiClient.approvalFlowService.Delete({ id: record.id });
    notification.success({ message: $t('ui.notification.delete_success') });
    await load();
  } catch {
    notification.error({ message: $t('ui.notification.delete_failed') });
  }
}

function addStep() {
  stepRows.value.push({ name: '', roleCode: '' });
}

function removeStep(index: number) {
  stepRows.value.splice(index, 1);
}

async function handleSave() {
  if (!form.value.bizType || !form.value.name) {
    notification.warning({ message: $t('page.approvalFlow.formInvalid') });
    return;
  }
  const steps = stepRows.value.filter((r) => r.roleCode);
  if (steps.length === 0) {
    notification.warning({ message: $t('page.approvalFlow.stepsRequired') });
    return;
  }

  saving.value = true;
  try {
    const payload = {
      bizType: form.value.bizType,
      name: form.value.name,
      steps: steps.map((r) => ({ name: r.name, roleCode: r.roleCode })),
    };
    if (editingId.value) {
      await apiClient.approvalFlowService.Update({
        data: { id: editingId.value, ...payload },
      } as any);
    } else {
      await apiClient.approvalFlowService.Create({ data: payload } as any);
    }
    notification.success({ message: $t('ui.notification.operation_success') });
    modalOpen.value = false;
    await load();
  } catch (e: any) {
    notification.error({
      message: e?.message || $t('ui.notification.operation_failed'),
    });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <Page :title="$t('page.approvalFlow.moduleName')">
    <div class="p-2">
      <div class="mb-2 flex justify-end gap-2">
        <a-button type="primary" @click="handleCreate">
          {{ $t('page.approvalFlow.button.create') }}
        </a-button>
      </div>

      <a-table
        :columns="columns"
        :data-source="flows"
        :loading="loading"
        :pagination="false"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'bizType'">
            {{ flowBizTypeToName(record.bizType) }}
          </template>
          <template v-else-if="column.key === 'stepsCount'">
            {{ (record.steps ?? []).length }}
          </template>
          <template v-else-if="column.key === 'steps'">
            {{ stepsText(record) }}
          </template>
          <template v-else-if="column.key === 'op'">
            <div class="flex gap-1">
              <a-button size="small" @click="handleEdit(record)">
                {{ $t('ui.button.edit') }}
              </a-button>
              <a-popconfirm
                :title="$t('ui.text.do_you_want_delete', { moduleName: $t('page.approvalFlow.moduleName') })"
                @confirm="handleDelete(record)"
              >
                <a-button danger size="small">
                  {{ $t('ui.button.delete') }}
                </a-button>
              </a-popconfirm>
            </div>
          </template>
        </template>
      </a-table>
    </div>

    <a-modal
      v-model:open="modalOpen"
      :title="
        editingId
          ? $t('ui.modal.edit', { moduleName: $t('page.approvalFlow.moduleName') })
          : $t('ui.modal.create', { moduleName: $t('page.approvalFlow.moduleName') })
      "
      :confirm-loading="saving"
      width="560px"
      @ok="handleSave"
    >
      <div class="flex flex-col gap-3 pt-2">
        <div class="flex items-center gap-2">
          <span class="w-24 shrink-0">{{ $t('page.approvalFlow.bizTypeLabel') }}</span>
          <a-select
            v-model:value="form.bizType"
            :options="bizTypeOptions"
            :disabled="!!editingId"
            class="flex-1"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-24 shrink-0">{{ $t('page.approvalFlow.name') }}</span>
          <a-input v-model:value="form.name" class="flex-1" />
        </div>

        <div>
          <div class="mb-2 text-sm font-medium">
            {{ $t('page.approvalFlow.steps') }}
          </div>
          <div
            v-for="(row, index) in stepRows"
            :key="index"
            class="mb-2 flex items-center gap-2"
          >
            <span class="w-8 shrink-0 text-gray-400">{{ index + 1 }}.</span>
            <a-input
              v-model:value="row.name"
              :placeholder="$t('page.approvalFlow.stepName')"
              style="width: 35%"
            />
            <a-select
              v-model:value="row.roleCode"
              :options="roleOptions"
              :placeholder="$t('page.approvalFlow.stepRole')"
              show-search
              option-filter-prop="label"
              class="flex-1"
            />
            <a-button
              danger
              size="small"
              :disabled="stepRows.length <= 1"
              @click="removeStep(index)"
            >
              {{ $t('ui.button.delete') }}
            </a-button>
          </div>
          <a-button @click="addStep">
            {{ $t('page.approvalFlow.addStep') }}
          </a-button>
        </div>
      </div>
    </a-modal>
  </Page>
</template>
