import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:meta/meta.dart';

import 'package:go_wind_erp/src/features/procurement/domain/procurement_failure.dart';
import 'package:go_wind_erp/src/features/procurement/domain/procurement_models.dart';
import 'package:go_wind_erp/src/features/procurement/domain/procurement_repository.dart';

/// 采购单详情视图状态。
@immutable
sealed class PurchaseOrderDetailState extends Equatable {
  const PurchaseOrderDetailState();
}

final class PurchaseOrderDetailLoading extends PurchaseOrderDetailState {
  const PurchaseOrderDetailLoading();

  @override
  List<Object?> get props => [];
}

final class PurchaseOrderDetailReady extends PurchaseOrderDetailState {
  final PurchaseOrderInfo order;

  const PurchaseOrderDetailReady({required this.order});

  @override
  List<Object?> get props => [order];
}

final class PurchaseOrderDetailFailure extends PurchaseOrderDetailState {
  final ProcurementFailure failure;

  const PurchaseOrderDetailFailure(this.failure);

  @override
  List<Object?> get props => [failure];
}

/// 采购单详情视图模型（只读，审批深链目标）。
class PurchaseOrderDetailCubit extends Cubit<PurchaseOrderDetailState> {
  final ProcurementRepository _repository;

  PurchaseOrderDetailCubit(this._repository)
      : super(const PurchaseOrderDetailLoading());

  Future<void> load(int id) async {
    emit(const PurchaseOrderDetailLoading());
    try {
      final order = await _repository.getPurchaseOrder(id);
      if (!isClosed) emit(PurchaseOrderDetailReady(order: order));
    } on ProcurementFailure catch (e) {
      if (!isClosed) emit(PurchaseOrderDetailFailure(e));
    }
  }
}
