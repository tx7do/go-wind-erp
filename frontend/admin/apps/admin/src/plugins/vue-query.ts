import type { App } from 'vue';
import { defineAsyncComponent } from 'vue';

import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';

/** 全局 QueryClient 实例，供 hooks 外部（Store、路由守卫等）调用 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 列表/详情查询一律不缓存：staleTime=0 使数据立即过期、每次挂载都打后端；
      // gcTime=0 在查询失去订阅时立刻回收缓存对象，避免保留默认 5 分钟的历史数据。
      staleTime: 0,
      gcTime: 0,
      retry: false,
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
    },
  },
});

/** Vue Query Devtools 组件（仅开发环境加载，生产环境为 null） */
export const TanstackQueryDevtools = import.meta.env.DEV
  ? defineAsyncComponent(async () => {
      const m = await import('@tanstack/vue-query-devtools');
      return m.VueQueryDevtools;
    })
  : null;

export function setupVueQuery(app: App) {
  app.use(VueQueryPlugin, {
    queryClient,
  });
}
