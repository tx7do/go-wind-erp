/**
 * 通用单据打印：将完整 HTML 文档写入隐藏 iframe 后调用浏览器打印。
 *
 * 走 iframe 而非 window.print()，是为了让打印内容脱离后台布局
 * （侧边栏/顶栏/@media 规则），Chrome 打印对话框可直接"另存为 PDF"。
 * iframe 在 afterprint 事件后移除；对 print() 无效的环境（无头/自动化）
 * 由 60s 兜底定时器回收。
 */

const DOC_CSS = `
@page { size: A4; margin: 14mm 12mm; }
* { box-sizing: border-box; }
body {
  font-family: 'Songti SC', 'SimSun', 'Noto Serif CJK SC', serif;
  color: #000;
  font-size: 12px;
  margin: 0;
}
.doc { padding: 8px 6px; }
.doc-title {
  text-align: center;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: 8px;
  margin: 0 0 4px;
}
.doc-sub {
  text-align: right;
  color: #444;
  font-size: 11px;
  margin: 0 0 10px;
}
.doc-info {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 10px;
  font-size: 12px;
}
.doc-info td {
  padding: 3px 4px;
  vertical-align: top;
  word-break: break-all;
}
.doc-items {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.doc-items th,
.doc-items td {
  border: 1px solid #333;
  padding: 5px 6px;
}
.doc-items th {
  background: #f0f0f0;
  font-weight: 600;
  white-space: nowrap;
}
.doc-items .num { text-align: right; white-space: nowrap; }
.doc-total {
  margin-top: 8px;
  font-size: 12px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
}
.doc-total .amount { font-size: 14px; font-weight: 600; white-space: nowrap; }
.doc-remark { margin-top: 8px; font-size: 12px; word-break: break-all; }
.doc-sign {
  margin-top: 30px;
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.doc-sign td { padding: 6px 4px; white-space: nowrap; }
`;

function escapeHtml(text: string): string {
  return text
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

/**
 * 打印一份完整 HTML 单据。
 * @param title 文档标题（同时作为浏览器打印任务名）
 * @param bodyHtml body 内的文档片段（使用 .doc-* 系列样式类）
 */
export function printHtml(title: string, bodyHtml: string): void {
  const iframe = document.createElement('iframe');
  iframe.setAttribute('aria-hidden', 'true');
  iframe.style.position = 'fixed';
  iframe.style.inset = 'auto 0 0 auto';
  iframe.style.width = '0';
  iframe.style.height = '0';
  iframe.style.border = '0';
  document.body.append(iframe);

  const cleanup = () => iframe.remove();
  const doc = iframe.contentDocument;
  if (!doc) {
    cleanup();
    return;
  }

  doc.open();
  doc.write(
    `<!DOCTYPE html><html><head><meta charset="utf-8"><title>${escapeHtml(title)}</title>` +
      `<style>${DOC_CSS}</style></head><body class="doc-print">${bodyHtml}</body></html>`,
  );
  doc.close();

  iframe.contentWindow?.addEventListener('afterprint', cleanup);
  // 无头/自动化环境下 print() 可能是空操作，afterprint 不会触发，用定时器兜底回收。
  window.setTimeout(cleanup, 60_000);
  window.setTimeout(() => {
    iframe.contentWindow?.focus();
    iframe.contentWindow?.print();
  }, 100);
}

export { escapeHtml };
