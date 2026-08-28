import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:go_wind_erp/src/features/wms/domain/wms_failure.dart';
import 'package:go_wind_erp/src/features/wms/domain/wms_models.dart';
import 'package:go_wind_erp/src/features/wms/domain/wms_repository.dart';
import 'package:go_wind_erp/src/features/wms/presentation/wms_state.dart';

/// WMS 扫码视图模型。
///
/// 仅依赖 [WmsRepository] 抽象。动作：
/// - [load]：进入页面拉取仓库列表。
/// - [selectWarehouse]：切换作业仓库，清空当前查询上下文。
/// - [lookupInventory]：按仓库 + productCode 查库存量，并联动拉取近期拣货单。
/// - [submitInternalTransfer]：发起 INTERNAL 调拨拣货单
///   （create → confirm → validate）。客户端守卫：目的仓须不同于源仓、
///   数量须为正且不超当前库存。入库拣货单不再由客户端发起——服务层在
///   采购单审批通过后自动生成。
/// - [clearMessage]：UI 展示完一次性提示后清除。
class WmsCubit extends Cubit<WmsState> {
  final WmsRepository _repository;

  WmsCubit(this._repository) : super(const WmsInitial());

  Future<void> load() async {
    emit(const WmsLoading());
    try {
      final warehouses = await _repository.listWarehouses();
      emit(
        WmsReady(
          warehouses: warehouses,
          selectedWarehouseCode:
              warehouses.isEmpty ? null : warehouses.first.code,
        ),
      );
    } on WmsFailure catch (e) {
      emit(WmsLoadFailure(e));
    }
  }

  void selectWarehouse(String code) {
    final s = state;
    if (s is! WmsReady) return;
    emit(
      s.copyWith(
        selectedWarehouseCode: code,
        inventory: null,
        lookupMiss: false,
        currentSku: '',
        pickings: const [],
        message: null,
      ),
    );
  }

  Future<void> lookupInventory(String productCode) async {
    final s = state;
    if (s is! WmsReady) return;
    final warehouse = s.selectedWarehouseCode;
    final product = productCode.trim();
    if (warehouse == null || product.isEmpty) return;

    emit(s.copyWith(lookingUp: true, lookupMiss: false, message: null));
    try {
      final inv = await _repository.findInventory(warehouse, product);
      if (isClosed) return;
      final cur = state as WmsReady;
      if (inv == null) {
        emit(
          cur.copyWith(
            lookingUp: false,
            inventory: null,
            lookupMiss: true,
            currentSku: product,
            pickings: const [],
          ),
        );
        return;
      }
      // 命中后联动近期拣货单；拉取失败不阻塞查询结果展示。
      var pickings = const <PickingRecord>[];
      try {
        pickings = await _repository.listPickings();
      } on WmsFailure {
        // 忽略：仅历史区为空。
      }
      if (isClosed) return;
      final cur2 = state as WmsReady;
      emit(
        cur2.copyWith(
          lookingUp: false,
          inventory: inv,
          lookupMiss: false,
          currentSku: product,
          pickings: pickings,
        ),
      );
    } on WmsFailure {
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(lookingUp: false, message: 'lookupFailed'));
    }
  }

  /// 内部调拨：源仓=当前选中仓库，productCode=当前查询上下文。
  /// 客户端守卫：目的仓须不同于源仓、数量须为正且不超当前库存
  /// （后端 create→confirm→validate 仍有防负库存守卫）。
  Future<void> submitInternalTransfer({
    required String toWarehouseCode,
    required int quantity,
  }) async {
    final s = state;
    if (s is! WmsReady) return;
    final from = s.selectedWarehouseCode;
    final inv = s.inventory;
    if (from == null || inv == null) return;

    if (toWarehouseCode == from) {
      emit(s.copyWith(message: 'sameWarehouse'));
      return;
    }
    if (quantity <= 0 || quantity > inv.quantity) {
      emit(s.copyWith(message: 'negativeStock'));
      return;
    }

    final draft = InternalTransferDraft(
      fromWarehouseCode: from,
      toWarehouseCode: toWarehouseCode,
      productCode: inv.productCode,
      plannedQuantity: quantity,
    );

    emit(s.copyWith(submitting: true, message: null));
    try {
      await _repository.submitInternalTransfer(draft);
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'transferSuccess'));
      await lookupInventory(inv.productCode);
    } on WmsFailure {
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'transferFailed'));
    }
  }

  /// 盘点：仓库=当前选中仓库，productCode=当前查询上下文。
  /// 客户端守卫：差异数非 0（后端按符号推导方向并有守卫；盘亏绝对值
  /// 不得超当前库存由防负守卫兜底）。
  Future<void> submitStocktake({required int diff}) async {
    final s = state;
    if (s is! WmsReady) return;
    final warehouse = s.selectedWarehouseCode;
    final inv = s.inventory;
    if (warehouse == null || inv == null) return;

    if (diff == 0) {
      emit(s.copyWith(message: 'stocktakeZeroDiff'));
      return;
    }
    if (diff < 0 && -diff > inv.quantity) {
      emit(s.copyWith(message: 'negativeStock'));
      return;
    }

    final draft = StocktakeDraft(
      warehouseCode: warehouse,
      productCode: inv.productCode,
      diff: diff,
    );

    emit(s.copyWith(submitting: true, message: null));
    try {
      await _repository.submitStocktake(draft);
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'stocktakeSuccess'));
      await lookupInventory(inv.productCode);
    } on WmsFailure {
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'stocktakeFailed'));
    }
  }

  void clearMessage() {
    final s = state;
    if (s is! WmsReady) return;
    emit(s.copyWith(message: null));
  }
}
