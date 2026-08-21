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
/// - [lookupInventory]：按仓库 + SKU 查库存，并联动拉取该组合的近期流水。
/// - [submitMovement]：提交出入库流水。出库做客户端守卫（不允许超过当前
///   库存导致负库存）；后端仍强校验 `before + delta == after`。
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
        movements: const [],
        message: null,
      ),
    );
  }

  Future<void> lookupInventory(String skuCode) async {
    final s = state;
    if (s is! WmsReady) return;
    final warehouse = s.selectedWarehouseCode;
    final sku = skuCode.trim();
    if (warehouse == null || sku.isEmpty) return;

    emit(s.copyWith(lookingUp: true, lookupMiss: false, message: null));
    try {
      final inv = await _repository.findInventory(warehouse, sku);
      if (isClosed) return;
      final cur = state as WmsReady;
      if (inv == null) {
        emit(
          cur.copyWith(
            lookingUp: false,
            inventory: null,
            lookupMiss: true,
            currentSku: sku,
            movements: const [],
          ),
        );
        return;
      }
      // 命中后联动近期流水；流水拉取失败不阻塞查询结果展示。
      var movements = const <StockMovementRecord>[];
      try {
        movements = await _repository.listMovements(warehouse, sku);
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
          currentSku: sku,
          movements: movements,
        ),
      );
    } on WmsFailure {
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(lookingUp: false, message: 'lookupFailed'));
    }
  }

  /// 提交流水。[quantity] 恒为正数，方向由 [kind] 决定。
  Future<void> submitMovement({
    required MovementKind kind,
    required int quantity,
    String? remark,
  }) async {
    final s = state;
    if (s is! WmsReady) return;
    final inv = s.inventory;
    if (inv == null) return;

    final delta = kind == MovementKind.outbound ? -quantity : quantity;

    // 客户端守卫：出库不允许超过当前库存（后端状态机不拦截负库存，
    // 负库存一旦入库账需靠反向流水冲正，代价高）。
    if (inv.quantity + delta < 0) {
      emit(s.copyWith(message: 'negativeStock'));
      return;
    }

    final draft = StockMovementDraft(
      warehouseCode: inv.warehouseCode,
      skuCode: inv.skuCode,
      kind: kind,
      delta: delta,
      quantityBefore: inv.quantity,
      quantityAfter: inv.quantity + delta,
      remark: (remark ?? '').isEmpty ? null : remark,
    );

    emit(s.copyWith(submitting: true, message: null));
    try {
      await _repository.submitMovement(draft);
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'submitSuccess'));
      // 提交成功后刷新库存与流水。
      await lookupInventory(inv.skuCode);
    } on WmsFailure {
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'submitFailed'));
    }
  }

  /// 库存调拨：源仓=当前选中仓库，SKU=当前查询上下文。
  /// 客户端守卫：目的仓须不同于源仓、数量须为正且不超当前库存
  /// （后端单事务原子执行，仍有防负库存守卫）。
  Future<void> transferStock({
    required String toWarehouseCode,
    required int quantity,
    String? remark,
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

    emit(s.copyWith(submitting: true, message: null));
    try {
      await _repository.transferStock(
        fromWarehouseCode: from,
        toWarehouseCode: toWarehouseCode,
        skuCode: inv.skuCode,
        quantity: quantity,
        remark: remark,
      );
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'transferSuccess'));
      await lookupInventory(inv.skuCode);
    } on WmsFailure {
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'transferFailed'));
    }
  }

  /// 冲正流水：等量反向台账，成功后刷新库存与流水。
  Future<void> reverseMovement(int movementId, String reason) async {
    final s = state;
    if (s is! WmsReady) return;

    emit(s.copyWith(submitting: true, message: null));
    try {
      await _repository.reverseMovement(movementId, reason);
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'reverseSuccess'));
      final sku = cur.currentSku;
      if (sku.isNotEmpty) {
        await lookupInventory(sku);
      }
    } on WmsFailure {
      if (isClosed) return;
      final cur = state as WmsReady;
      emit(cur.copyWith(submitting: false, message: 'reverseFailed'));
    }
  }

  void clearMessage() {
    final s = state;
    if (s is! WmsReady) return;
    emit(s.copyWith(message: null));
  }
}
