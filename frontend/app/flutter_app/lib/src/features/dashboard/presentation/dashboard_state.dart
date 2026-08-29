import 'package:equatable/equatable.dart';
import 'package:meta/meta.dart';

import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_failure.dart';
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_models.dart';

/// 看板视图状态。
@immutable
sealed class DashboardState extends Equatable {
  const DashboardState();
}

final class DashboardInitial extends DashboardState {
  const DashboardInitial();

  @override
  List<Object?> get props => [];
}

final class DashboardLoading extends DashboardState {
  const DashboardLoading();

  @override
  List<Object?> get props => [];
}

final class DashboardReady extends DashboardState {
  final InventoryOverviewInfo overview;

  /// 财务汇总（本月收入/成本/利润 + 应收应付）。拉取失败时为 null——
  /// 库存看板不因此不可用。
  final FinanceSummaryInfo? finance;

  const DashboardReady(this.overview, {this.finance});

  @override
  List<Object?> get props => [overview, finance];
}

final class DashboardFailureState extends DashboardState {
  final DashboardFailure failure;

  const DashboardFailureState(this.failure);

  @override
  List<Object?> get props => [failure];
}
