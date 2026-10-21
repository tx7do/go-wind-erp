/**
 * 表格导出工具（TableExportButton 的依赖）。
 *
 * 与上游 utils/csv.ts 同 API（downloadTableFile + CsvColumn + TableExportFormat），
 * 实现差异：XLSX 用仓内既有 SheetJS（xlsx）依赖生成，而不是上游的 exceljs——
 * 下游清单没有 exceljs，不为本文件单加依赖。CSV 手拼 + BOM 与上游一致。
 */

import * as XLSX from 'xlsx';

export interface CsvColumn {
  title: string;
  key: string;
}

export type TableExportFormat = 'csv' | 'xlsx';

function escapeCsvCell(value: unknown): string {
  if (value === null || value === undefined) return '';
  const str = String(value);
  if (/[",\n\r]/.test(str)) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export function downloadCsv(
  filename: string,
  columns: CsvColumn[],
  rows: any[],
): void {
  const header = columns.map((c) => escapeCsvCell(c.title)).join(',');
  const lines = rows.map((row) =>
    columns.map((c) => escapeCsvCell((row as any)?.[c.key])).join(','),
  );
  // BOM：保证 Excel 打开中文不乱码
  const content = `\uFEFF${header}\n${lines.join('\n')}`;
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8' });
  triggerDownload(blob, filename);
}

async function downloadXlsx(
  filename: string,
  columns: CsvColumn[],
  rows: any[],
): Promise<void> {
  const ws = XLSX.utils.aoa_to_sheet([
    columns.map((c) => c.title),
    ...rows.map((row) => columns.map((c) => (row as any)?.[c.key] ?? '')),
  ]);
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, 'Sheet1');
  XLSX.writeFile(wb, `${filename}.xlsx`);
}

/**
 * 按格式分发导出下载。传入文件名不带扩展名，由本函数按格式追加。
 */
export async function downloadTableFile(
  format: TableExportFormat,
  filename: string,
  columns: CsvColumn[],
  rows: any[],
): Promise<void> {
  if (format === 'xlsx') {
    await downloadXlsx(filename, columns, rows);
    return;
  }
  downloadCsv(`${filename}.csv`, columns, rows);
}
