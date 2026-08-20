import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:meta/meta.dart';

import 'package:go_wind_erp/src/features/approval/domain/approval_failure.dart';
import 'package:go_wind_erp/src/features/approval/domain/approval_models.dart';
import 'package:go_wind_erp/src/features/approval/domain/approval_repository.dart';

/// 审批视图状态。
@immutable
sealed class ApprovalState extends Equatable {
  const ApprovalState();
}

final class ApprovalInitial extends ApprovalState {
  const ApprovalInitial();

  @override
  List<Object?> get props => [];
}

final class ApprovalLoading extends ApprovalState {
  const ApprovalLoading();

  @override
  List<Object?> get props => [];
}

final class ApprovalReady extends ApprovalState {
  final ApprovalFilter filter;
  final List<ApprovalInfo> requests;

  /// 动作（通过/驳回/撤销）执行中，锁定相应按钮。
  final bool acting;

  const ApprovalReady({
    required this.filter,
    required this.requests,
    this.acting = false,
  });

  ApprovalReady copyWith({
    ApprovalFilter? filter,
    List<ApprovalInfo>? requests,
    bool? acting,
  }) {
    return ApprovalReady(
      filter: filter ?? this.filter,
      requests: requests ?? this.requests,
      acting: acting ?? this.acting,
    );
  }

  @override
  List<Object?> get props => [filter, requests, acting];
}

final class ApprovalFailureState extends ApprovalState {
  final ApprovalFailure failure;

  const ApprovalFailureState(this.failure);

  @override
  List<Object?> get props => [failure];
}

/// 审批中心视图模型。
///
/// 仅依赖 [ApprovalRepository] 抽象。[setFilter] 切换状态筛选并重新拉取；
/// [approve]/[reject]/[cancel] 执行后刷新当前列表。动作的业务拒绝
/// （状态机/所有权）不整页报错，经 [ApprovalReady] 上层由 UI 以 SnackBar
/// 提示——因此动作失败时 emit 回 Ready（列表保持不变）。
class ApprovalCubit extends Cubit<ApprovalState> {
  final ApprovalRepository _repository;

  ApprovalCubit(this._repository) : super(const ApprovalInitial());

  Future<void> load(ApprovalFilter filter) async {
    emit(const ApprovalLoading());
    await _fetch(filter);
  }

  Future<void> setFilter(ApprovalFilter filter) async {
    final s = state;
    if (s is ApprovalReady) {
      emit(s.copyWith(filter: filter));
    }
    await _fetch(filter);
  }

  Future<void> refresh() async {
    final s = state;
    await _fetch(s is ApprovalReady ? s.filter : ApprovalFilter.all);
  }

  Future<void> _fetch(ApprovalFilter filter) async {
    try {
      final requests = await _repository.listRequests(filter);
      if (!isClosed) {
        emit(ApprovalReady(filter: filter, requests: requests));
      }
    } on ApprovalFailure catch (e) {
      if (!isClosed) emit(ApprovalFailureState(e));
    }
  }

  /// 执行审批动作。[onRejected] 在业务拒绝时被回调（由 UI 展示提示），
  /// 列表保持不变；成功则刷新列表。
  Future<void> act(
    Future<void> Function() action, {
    void Function(ApprovalFailure)? onRejected,
  }) async {
    final s = state;
    if (s is! ApprovalReady) return;
    emit(s.copyWith(acting: true));
    try {
      await action();
      if (isClosed) return;
      final cur = state as ApprovalReady;
      emit(cur.copyWith(acting: false));
      await refresh();
    } on ApprovalFailure catch (e) {
      if (isClosed) return;
      final cur = state as ApprovalReady;
      emit(cur.copyWith(acting: false));
      onRejected?.call(e);
    }
  }

  Future<void> approve(
    int id, {
    String? comment,
    void Function(ApprovalFailure)? onRejected,
  }) =>
      act(() => _repository.approve(id, comment: comment),
          onRejected: onRejected);

  Future<void> reject(
    int id, {
    String? comment,
    void Function(ApprovalFailure)? onRejected,
  }) =>
      act(() => _repository.reject(id, comment: comment),
          onRejected: onRejected);

  Future<void> cancel(int id, {void Function(ApprovalFailure)? onRejected}) =>
      act(() => _repository.cancel(id), onRejected: onRejected);
}
