<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import type { Recordable } from '@vben/types';

import { computed, h, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

import { AuthenticationRegister, z } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { fetchGenerateCaptcha } from '#/api/composables';
import { apiClient } from '#/api/client';
import { setCaptchaHeaders } from '#/transport/rest';

defineOptions({ name: 'Register' });

const router = useRouter();
const loading = ref(false);

// 验证码状态（与登录页同机制：注册是公开端点，后端强制校验验证码）
const captchaId = ref('');
const captchaImage = ref('');
const captchaLoading = ref(false);

async function refreshCaptcha() {
  captchaLoading.value = true;
  try {
    const resp = await fetchGenerateCaptcha();
    captchaId.value = resp.captchaId ?? '';
    captchaImage.value = resp.imageBase64 ?? '';
  } catch {
    // 验证码获取失败不阻断页面
  } finally {
    captchaLoading.value = false;
  }
}

onMounted(() => {
  refreshCaptcha();
});

async function handleSubmit(values: Recordable<any>) {
  loading.value = true;
  try {
    if (captchaId.value && values.captchaValue) {
      setCaptchaHeaders(captchaId.value, values.captchaValue);
    }

    await apiClient.tenantService.SelfRegisterTenant({
      tenantName: values.tenantName,
      tenantCode: values.tenantCode,
      adminUsername: values.adminUsername,
      password: values.password,
    });

    notification.success({
      message: $t('page.register.successTitle'),
      description: $t('page.register.successDesc', {
        tenantCode: values.tenantCode,
        username: values.adminUsername,
      }),
      duration: 6,
    });

    // 注册成功跳登录页，并预填租户编码方便立即登录
    await router.push({
      path: '/auth/login',
    });
  } catch {
    refreshCaptcha();
  } finally {
    loading.value = false;
  }
}

const renderCaptchaImage = () =>
  h(
    'div',
    {
      title: $t('authentication.captchaRefresh'),
      onClick: () => {
        if (!captchaLoading.value) refreshCaptcha();
      },
      style: {
        height: '36px',
        width: '110px',
        flexShrink: '0',
        cursor: 'pointer',
        borderRadius: '6px',
        overflow: 'hidden',
        border: '1px solid #d9d9d9',
        background: '#f5f5f5',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      },
    },
    captchaImage.value
      ? [
          h('img', {
            src: captchaImage.value,
            alt: 'captcha',
            style: {
              height: '100%',
              width: '100%',
              objectFit: 'cover',
            },
          }),
        ]
      : [
          h(
            'span',
            { style: { color: '#999', fontSize: '12px' } },
            captchaLoading.value ? '...' : $t('authentication.captchaRefresh'),
          ),
        ],
  );

const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('page.register.tenantNameTip'),
      },
      fieldName: 'tenantName',
      label: $t('page.register.tenantName'),
      rules: z.string().min(1, { message: $t('page.register.tenantNameTip') }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('page.register.tenantCodeTip'),
      },
      fieldName: 'tenantCode',
      label: $t('page.register.tenantCode'),
      rules: z
        .string()
        .min(2, { message: $t('page.register.tenantCodeTip') })
        .regex(/^[a-zA-Z0-9_-]+$/, {
          message: $t('page.register.tenantCodeInvalid'),
        }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('authentication.usernameTip'),
      },
      fieldName: 'adminUsername',
      label: $t('page.register.adminUsername'),
      rules: z.string().min(1, { message: $t('authentication.usernameTip') }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        passwordStrength: true,
        placeholder: $t('authentication.password'),
      },
      fieldName: 'password',
      label: $t('authentication.password'),
      renderComponentContent() {
        return {
          strengthText: () => $t('authentication.passwordStrength'),
        };
      },
      rules: z.string().min(6, { message: $t('page.register.passwordMin') }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: $t('authentication.confirmPassword'),
      },
      dependencies: {
        rules(values) {
          const { password } = values;
          return z
            .string({ required_error: $t('authentication.passwordTip') })
            .min(1, { message: $t('authentication.passwordTip') })
            .refine((value) => value === password, {
              message: $t('authentication.confirmPasswordTip'),
            });
        },
        triggerFields: ['password'],
      },
      fieldName: 'confirmPassword',
      label: $t('authentication.confirmPassword'),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('authentication.captchaTip'),
        autocomplete: 'off',
        class: 'w-auto flex-1 min-w-0',
      },
      fieldName: 'captchaValue',
      label: $t('authentication.captcha'),
      rules: z.string().min(1, { message: $t('authentication.captchaTip') }),
      suffix: renderCaptchaImage,
    },
    {
      component: 'VbenCheckbox',
      fieldName: 'agreePolicy',
      renderComponentContent: () => ({
        default: () =>
          h('span', [
            $t('authentication.agree'),
            h(
              'a',
              {
                class: 'vben-link ml-1 ',
                href: '',
              },
              `${$t('authentication.privacyPolicy')} & ${$t('authentication.terms')}`,
            ),
          ]),
      }),
      rules: z.boolean().refine((value) => !!value, {
        message: $t('authentication.agreeTip'),
      }),
    },
  ];
});
</script>

<template>
  <AuthenticationRegister
    :form-schema="formSchema"
    :loading="loading"
    @submit="handleSubmit"
  />
</template>
