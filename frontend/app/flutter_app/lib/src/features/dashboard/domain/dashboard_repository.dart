import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_models.dart';

/// 看板仓储抽象。
///
/// presentation 层仅依赖本接口；实现见 data 层 [DashboardRepositoryImpl]。
abstract class DashboardRepository {
  /// 拉取库存经营总览。
  ///
  /// [lowStockThreshold] 低库存阈值（默认 10）；[lowStockLimit] 清单条数上限
  /// （默认 10）。
  Future<InventoryOverviewInfo> fetchOverview({
    int? lowStockThreshold,
    int? lowStockLimit,
  });

  /// 拉取财务经营汇总（本月收入/成本/利润 + 应收应付余额）。
  Future<FinanceSummaryInfo> fetchFinanceSummary();
}
