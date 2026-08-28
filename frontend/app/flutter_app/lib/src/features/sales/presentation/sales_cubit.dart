import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:meta/meta.dart';

import 'package:go_wind_erp/src/features/sales/domain/sales_failure.dart';
import 'package:go_wind_erp/src/features/sales/domain/sales_models.dart';
import 'package:go_wind_erp/src/features/sales/domain/sales_repository.dart';

/// 销售单列表筛选。
enum SalesFilter {
  all,
  draft,
  submitted,
  approved,
  completed,
  cancelled;

  /// 后端 status 过滤值；null 表示全部。
  String? get statusValue => switch (this) {
        SalesFilter.all => null,
        SalesFilter.draft => 'DRAFT',
        SalesFilter.submitted => 'SUBMITTED',
        SalesFilter.approved => 'APPROVED',
        SalesFilter.completed => 'COMPLETED',
        SalesFilter.cancelled => 'CANCELLED',
      };
}

/// 销售单列表视图状态。
@immutable
sealed class SalesState extends Equatable {
  const SalesState();
}

final class SalesInitial extends SalesState {
  const SalesInitial();

  @override
  List<Object?> get props => [];
}

final class SalesLoading extends SalesState {
  const SalesLoading();

  @override
  List<Object?> get props => [];
}

final class SalesReady extends SalesState {
  final SalesFilter filter;
  final List<SalesOrderInfo> orders;

  const SalesReady({
    required this.filter,
    required this.orders,
  });

  SalesReady copyWith({
    SalesFilter? filter,
    List<SalesOrderInfo>? orders,
  }) {
    return SalesReady(
      filter: filter ?? this.filter,
      orders: orders ?? this.orders,
    );
  }

  @override
  List<Object?> get props => [filter, orders];
}

final class SalesFailureState extends SalesState {
  final SalesFailure failure;

  const SalesFailureState(this.failure);

  @override
  List<Object?> get props => [failure];
}

/// 销售单详情视图状态（独立于列表，由详情页持有）。
@immutable
sealed class SalesDetailState extends Equatable {
  const SalesDetailState();
}

final class SalesDetailLoading extends SalesDetailState {
  const SalesDetailLoading();

  @override
  List<Object?> get props => [];
}

final class SalesDetailReady extends SalesDetailState {
  final SalesOrderInfo order;

  /// 退货提交中，锁定相应按钮。
  final bool returning;

  /// 一次性消息（退货成功/失败提示），消费后调 [consumeMessage] 清空。
  final String? message;

  const SalesDetailReady({
    required this.order,
    this.returning = false,
    this.message,
  });

  SalesDetailReady copyWith({
    SalesOrderInfo? order,
    bool? returning,
    String? message,
    bool clearMessage = false,
  }) {
    return SalesDetailReady(
      order: order ?? this.order,
      returning: returning ?? this.returning,
      message: clearMessage ? null : (message ?? this.message),
    );
  }

  @override
  List<Object?> get props => [order, returning, message];
}

/// 销售单列表视图模型。
class SalesCubit extends Cubit<SalesState> {
  final SalesRepository _repository;

  SalesCubit(this._repository) : super(const SalesInitial());

  Future<void> load() => setFilter(SalesFilter.all);

  Future<void> setFilter(SalesFilter filter) async {
    final s = state;
    if (s is SalesReady) {
      emit(s.copyWith(filter: filter));
    } else {
      emit(const SalesLoading());
    }
    try {
      final orders = await _repository.listSalesOrders(
        status: filter.statusValue,
      );
      if (!isClosed) emit(SalesReady(filter: filter, orders: orders));
    } on SalesFailure catch (e) {
      if (!isClosed) emit(SalesFailureState(e));
    }
  }
}

/// 销售单详情视图模型。
class SalesDetailCubit extends Cubit<SalesDetailState> {
  final SalesRepository _repository;

  SalesDetailCubit(this._repository) : super(const SalesDetailLoading());

  Future<void> load(int id) async {
    emit(const SalesDetailLoading());
    try {
      final order = await _repository.getSalesOrder(id);
      if (!isClosed) emit(SalesDetailReady(order: order));
    } on SalesFailure {
      if (!isClosed) {
        emit(
          const SalesDetailReady(
            order: SalesOrderInfo(
              id: 0,
              soNumber: '',
              customerCode: '',
              warehouseCode: '',
              status: '',
              totalAmount: 0,
            ),
            message: 'loadFailed',
          ),
        );
      }
    }
  }

  /// 提交销售退货：成功后刷新详情（履约数回退需在拣货单确认+执行后发生，
  /// 此处提示用户后续步骤）；失败以 message 提示，页面保持。
  Future<void> submitReturn(SalesReturnDraft draft) async {
    final s = state;
    if (s is! SalesDetailReady) return;
    emit(s.copyWith(returning: true));
    try {
      await _repository.submitSalesReturn(draft);
      if (isClosed) return;
      final order = await _repository.getSalesOrder(draft.salesOrderId);
      if (!isClosed) {
        emit(
          SalesDetailReady(order: order, message: 'returnCreated'),
        );
      }
    } on SalesFailure {
      if (!isClosed) {
        final cur = state as SalesDetailReady;
        emit(cur.copyWith(returning: false, message: 'returnFailed'));
      }
    }
  }

  void consumeMessage() {
    final s = state;
    if (s is SalesDetailReady && s.message != null) {
      emit(s.copyWith(clearMessage: true));
    }
  }
}
