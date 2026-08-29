/**
 * 采购单/销售单打印文档构建。
 *
 * 打印内容为 A4 竖版单据：标题 + 单号/打印时间、主信息区、明细表、
 * 合计（含人民币大写）、备注与签署栏。品名/规格/单位等主数据在打印时
 * 从对应档案接口解析（列表行只带编码）。
 */

import { $t } from '@vben/locales';

import {
  apiClient,
  centsToYuan,
  fetchListCustomers,
  fetchListProducts,
  fetchListSuppliers,
  fetchListWarehouses,
  PaginationQuery,
  purchaseOrderStatusToName,
  salesOrderStatusToName,
  type procurementservicev1_PurchaseOrder as PurchaseOrder,
  type salesservicev1_SalesOrder as SalesOrder,
} from '#/api';

import { escapeHtml, printHtml } from './print';

/** 人民币金额大写（精确到分），如 1234.56 → 壹仟贰佰叁拾肆元伍角陆分。 */
export function amountToChinese(amountYuan: number): string {
  const digits = ['零', '壹', '贰', '叁', '肆', '伍', '陆', '柒', '捌', '玖'];
  const smallUnits = ['', '拾', '佰', '仟'];
  const bigUnits = ['', '万', '亿'];

  const cents = Math.round(amountYuan * 100);
  if (!Number.isFinite(cents)) return '';
  if (cents === 0) return '零元整';

  const negative = cents < 0;
  const abs = Math.abs(cents);
  const intPart = Math.floor(abs / 100);
  const jiao = Math.floor(abs / 10) % 10;
  const fen = abs % 10;

  let intText = '';
  if (intPart > 0) {
    const groupCount = Math.floor((String(intPart).length - 1) / 4) + 1;
    const groups: number[] = [];
    let n = intPart;
    for (let i = 0; i < groupCount; i++) {
      groups.unshift(n % 10_000);
      n = Math.floor(n / 10_000);
    }
    let pendingZero = false;
    for (const [idx, g] of groups.entries()) {
      const bigIdx = groupCount - 1 - idx;
      if (g === 0) {
        pendingZero = true;
        continue;
      }
      const s = String(g);
      let seg = '';
      let zero = false;
      for (let i = 0; i < s.length; i++) {
        const d = Number(s[i]);
        const u = s.length - 1 - i;
        if (d === 0) {
          if (seg !== '') zero = true;
        } else {
          seg += (zero ? '零' : '') + digits[d] + smallUnits[u];
          zero = false;
        }
      }
      // 高组已输出且本组不足千位（或前组为零）时补"零"，如 10亿零4万零5元。
      const prefixZero = intText !== '' && (pendingZero || g < 1000);
      intText += (prefixZero ? '零' : '') + seg + bigUnits[bigIdx];
      pendingZero = false;
    }
    intText += '元';
  }

  let decText = '';
  if (jiao === 0 && fen === 0) {
    decText = '整';
  } else {
    if (intPart > 0 && jiao === 0 && fen > 0) decText += '零';
    if (jiao > 0) decText += digits[jiao] + '角';
    if (fen > 0) decText += digits[fen] + '分';
  }

  return (negative ? '负' : '') + intText + decText;
}

interface DocField {
  label: string;
  value: string;
}

interface OrderDocParams {
  title: string;
  docNumber: string;
  infoRows: DocField[][];
  itemColumns: string[];
  numericCols: number[];
  itemRows: string[][];
  totalUpper: string;
  totalAmount: number;
  remark?: string;
  signLabels: string[];
}

function formatDateTime(iso: null | string | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

function buildOrderDocHtml(p: OrderDocParams): string {
  const infoHtml = p.infoRows
    .map(
      (row) =>
        `<tr>${row
          .map((f) => `<td>${escapeHtml(f.label)}：${escapeHtml(f.value)}</td>`)
          .join('')}</tr>`,
    )
    .join('');
  const headHtml = p.itemColumns
    .map((c) => `<th>${escapeHtml(c)}</th>`)
    .join('');
  const bodyHtml = p.itemRows
    .map(
      (row) =>
        `<tr>${row
          .map((cell, i) => {
            const cls = p.numericCols.includes(i) ? ' class="num"' : '';
            return `<td${cls}>${escapeHtml(cell)}</td>`;
          })
          .join('')}</tr>`,
    )
    .join('');
  const signHtml = p.signLabels
    .map((l) => `<td>${escapeHtml(l)}：</td>`)
    .join('');

  return `
<div class="doc">
  <h1 class="doc-title">${escapeHtml(p.title)}</h1>
  <div class="doc-sub">${escapeHtml($t('page.doc.docNumber'))}：${escapeHtml(p.docNumber)}　·　${escapeHtml($t('page.doc.printTime'))}：${formatDateTime(new Date().toISOString())}</div>
  <table class="doc-info">${infoHtml}</table>
  <table class="doc-items">
    <thead><tr>${headHtml}</tr></thead>
    <tbody>${bodyHtml}</tbody>
  </table>
  <div class="doc-total">
    <span>${escapeHtml($t('page.doc.totalUpper'))}：${escapeHtml(p.totalUpper)}</span>
    <span class="amount">¥ ${escapeHtml(centsToYuan(p.totalAmount))}</span>
  </div>
  ${p.remark ? `<div class="doc-remark">${escapeHtml($t('page.doc.remark'))}：${escapeHtml(p.remark)}</div>` : ''}
  <table class="doc-sign"><tr>${signHtml}</tr></table>
</div>`;
}

interface PartnerRef {
  code?: string;
  name?: string;
}

function partnerLabel(map: Map<string, PartnerRef>, code?: string): string {
  if (!code) return '—';
  const ref = map.get(code);
  return ref?.name ? `${code} — ${ref.name}` : code;
}

async function loadRefMap<T extends { code?: string }>(
  fetch: (
    params: PaginationQuery,
  ) => Promise<{ items?: T[] } | undefined | null>,
): Promise<Map<string, T>> {
  const result = await fetch(
    new PaginationQuery({ paging: { page: 1, pageSize: 500 } }),
  );
  return new Map((result?.items ?? []).map((it) => [it.code ?? '', it]));
}

/** 打印采购单（order 需带明细，通常来自 Get 接口）。 */
export async function printPurchaseOrder(order: PurchaseOrder): Promise<void> {
  const [suppliers, warehouses, products] = await Promise.all([
    loadRefMap(fetchListSuppliers),
    loadRefMap(fetchListWarehouses),
    loadRefMap(fetchListProducts),
  ]);
  const items = order.items ?? [];
  const rows = items.map((it) => {
    const p = products.get(it.skuCode ?? '');
    return [
      it.skuCode ?? '',
      p?.name ?? '—',
      p?.spec || '—',
      p?.unit || '—',
      String(it.quantity ?? 0),
      centsToYuan(it.unitPrice),
      centsToYuan(it.amount),
      String(it.receivedQuantity ?? 0),
    ];
  });
  const total =
    order.totalAmount ?? items.reduce((sum, it) => sum + (it.amount ?? 0), 0);
  const title = $t('page.doc.purchaseTitle');
  const html = buildOrderDocHtml({
    title,
    docNumber: order.poNumber ?? '',
    infoRows: [
      [
        {
          label: $t('page.doc.supplier'),
          value: partnerLabel(suppliers, order.supplierCode),
        },
        {
          label: $t('page.doc.warehouse'),
          value: partnerLabel(warehouses, order.warehouseCode),
        },
      ],
      [
        {
          label: $t('page.doc.status'),
          value: purchaseOrderStatusToName(order.status ?? 'DRAFT'),
        },
        {
          label: $t('ui.table.createdAt'),
          value: formatDateTime(order.createdAt),
        },
      ],
    ],
    itemColumns: [
      $t('page.doc.sku'),
      $t('page.doc.productName'),
      $t('page.doc.spec'),
      $t('page.doc.unit'),
      $t('page.doc.quantity'),
      $t('page.doc.unitPrice'),
      $t('page.doc.amount'),
      $t('page.doc.receivedQuantity'),
    ],
    numericCols: [4, 5, 6, 7],
    itemRows: rows,
    totalUpper: amountToChinese(total / 100),
    totalAmount: total,
    remark: order.remark,
    signLabels: [
      $t('page.doc.sign.creator'),
      $t('page.doc.sign.approver'),
      $t('page.doc.sign.supplierConfirm'),
      $t('page.doc.sign.receiver'),
    ],
  });
  printHtml(`${title} ${order.poNumber ?? ''}`, html);
}

/** 按主键打印采购单（列表行动作入口：先拉完整单据再打印）。 */
export async function printPurchaseOrderById(id: number): Promise<void> {
  const order = await apiClient.purchaseOrderService.Get({ id });
  if (order?.id) {
    await printPurchaseOrder(order);
  }
}

/** 打印销售单（order 需带明细，通常来自 Get 接口）。 */
export async function printSalesOrder(order: SalesOrder): Promise<void> {
  const [customers, warehouses, products] = await Promise.all([
    loadRefMap(fetchListCustomers),
    loadRefMap(fetchListWarehouses),
    loadRefMap(fetchListProducts),
  ]);
  const items = order.items ?? [];
  const rows = items.map((it) => {
    const p = products.get(it.skuCode ?? '');
    return [
      it.skuCode ?? '',
      p?.name ?? '—',
      p?.spec || '—',
      p?.unit || '—',
      String(it.quantity ?? 0),
      centsToYuan(it.unitPrice),
      centsToYuan(it.amount),
      String(it.fulfilledQuantity ?? 0),
    ];
  });
  const total =
    order.totalAmount ?? items.reduce((sum, it) => sum + (it.amount ?? 0), 0);
  const title = $t('page.doc.salesTitle');
  const html = buildOrderDocHtml({
    title,
    docNumber: order.soNumber ?? '',
    infoRows: [
      [
        {
          label: $t('page.doc.customer'),
          value: partnerLabel(customers, order.customerCode),
        },
        {
          label: $t('page.doc.warehouse'),
          value: partnerLabel(warehouses, order.warehouseCode),
        },
      ],
      [
        {
          label: $t('page.doc.status'),
          value: salesOrderStatusToName(order.status ?? 'DRAFT'),
        },
        {
          label: $t('ui.table.createdAt'),
          value: formatDateTime(order.createdAt),
        },
      ],
    ],
    itemColumns: [
      $t('page.doc.sku'),
      $t('page.doc.productName'),
      $t('page.doc.spec'),
      $t('page.doc.unit'),
      $t('page.doc.quantity'),
      $t('page.doc.unitPrice'),
      $t('page.doc.amount'),
      $t('page.doc.fulfilledQuantity'),
    ],
    numericCols: [4, 5, 6, 7],
    itemRows: rows,
    totalUpper: amountToChinese(total / 100),
    totalAmount: total,
    remark: order.remark,
    signLabels: [
      $t('page.doc.sign.creator'),
      $t('page.doc.sign.approver'),
      $t('page.doc.sign.customerConfirm'),
      $t('page.doc.sign.dispatcher'),
    ],
  });
  printHtml(`${title} ${order.soNumber ?? ''}`, html);
}

/** 按主键打印销售单（列表行动作入口：先拉完整单据再打印）。 */
export async function printSalesOrderById(id: number): Promise<void> {
  const order = await apiClient.salesOrderService.Get({ id });
  if (order?.id) {
    await printSalesOrder(order);
  }
}
