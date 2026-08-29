/**
 * 通用 Excel 导出（SheetJS）。
 *
 * 统一入口 [exportRowsToExcel]：表头 + 行数据二维数组 → 单工作表 xlsx。
 * 供报表页（余额表/凭证/对账单/销售排行）复用；文件名由调用方带日期后缀。
 */

import * as XLSX from 'xlsx';

export function exportRowsToExcel(
  filename: string,
  sheetName: string,
  headers: string[],
  rows: (number | string)[][],
): void {
  const ws = XLSX.utils.aoa_to_sheet([headers, ...rows]);
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, sheetName.slice(0, 30) || 'Sheet1');
  XLSX.writeFile(wb, filename.endsWith('.xlsx') ? filename : `${filename}.xlsx`);
}
