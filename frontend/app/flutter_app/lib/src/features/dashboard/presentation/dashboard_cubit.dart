import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_failure.dart';
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
      final overview = await _repository.fetchOverview();
      if (!isClosed) emit(DashboardReady(overview));
    } on DashboardFailure catch (e) {
      if (!isClosed) emit(DashboardFailureState(e));
    }
  }
}
