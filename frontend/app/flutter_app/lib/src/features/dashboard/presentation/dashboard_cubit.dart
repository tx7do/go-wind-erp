import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_failure.dart';
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_models.dart';
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_repository.dart';
import 'package:go_wind_erp/src/features/dashboard/presentation/dashboard_state.dart';

/// 看板视图模型。
///
/// 仅依赖 [DashboardRepository] 抽象。[load] 可重复调用（下拉刷新）。
class DashboardCubit extends Cubit<DashboardState> {
  final DashboardRepository _repository;

  DashboardCubit(this._repository) : super(const DashboardInitial());

  Future<void> load() async {
    emit(const DashboardLoading());
    try {
      // 财务汇总软失败：失败不阻塞库存看板（finance 置空）。
      final overview = await _repository.fetchOverview();
      FinanceSummaryInfo? finance;
      try {
        finance = await _repository.fetchFinanceSummary();
      } on DashboardFailure {
        finance = null;
      }
      if (!isClosed) {
        emit(DashboardReady(overview, finance: finance));
      }
    } on DashboardFailure catch (e) {
      if (!isClosed) emit(DashboardFailureState(e));
    }
  }
}
