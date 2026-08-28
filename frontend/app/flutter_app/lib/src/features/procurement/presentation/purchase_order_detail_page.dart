import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:get_it/get_it.dart';

import 'package:go_wind_erp/generated/l10n.dart';
import 'package:go_wind_erp/src/features/procurement/domain/procurement_repository.dart';
import 'package:go_wind_erp/src/features/procurement/presentation/procurement_cubit.dart';

/// 采购单详情页（移动端只读，审批深链目标）。
///
/// 单头信息 + 明细已收进度；采购退货在 Web 管理端操作，移动端不提供。
class PurchaseOrderDetailPage extends StatelessWidget {
  final int orderId;

  const PurchaseOrderDetailPage({super.key, required this.orderId});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return BlocProvider(
      create: (_) => PurchaseOrderDetailCubit(
        GetIt.instance<ProcurementRepository>(),
      )..load(orderId),
      child: Scaffold(
        appBar: AppBar(title: Text(loc.poDetailTitle)),
        body: _DetailBody(orderId: orderId),
      ),
    );
  }
}

class _DetailBody extends StatelessWidget {
  final int orderId;

  const _DetailBody({required this.orderId});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return BlocBuilder<PurchaseOrderDetailCubit, PurchaseOrderDetailState>(
      builder: (context, state) {
        if (state is PurchaseOrderDetailReady) {
          final order = state.order;
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        order.poNumber,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),
                      _kv(loc.poSupplier, order.supplierCode),
                      _kv(loc.salesWarehouse, order.warehouseCode),
                      _kv(
                        loc.salesTotal,
                        '¥${(order.totalAmount / 100).toStringAsFixed(2)}',
                      ),
                      _kv(loc.salesStatus, _statusLabel(loc, order.status)),
                      if ((order.remark ?? '').isNotEmpty)
                        _kv(loc.salesRemark, order.remark!),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 8),
              Text(loc.salesItems),
              const SizedBox(height: 4),
              for (final item in order.items)
                Card(
                  child: ListTile(
                    title: Text(item.skuCode),
                    subtitle: Text(
                      '${loc.salesQuantity}: ${item.quantity} · '
                      '${loc.poReceived}: ${item.receivedQuantity}',
                    ),
                  ),
                ),
              if (order.items.isEmpty)
                Padding(
                  padding: const EdgeInsets.all(24),
                  child: Center(child: Text(loc.salesEmpty)),
                ),
            ],
          );
        }
        if (state is PurchaseOrderDetailFailure) {
          return Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(loc.loadFailed),
                const SizedBox(height: 12),
                FilledButton(
                  onPressed: () =>
                      context.read<PurchaseOrderDetailCubit>().load(orderId),
                  child: Text(loc.retry),
                ),
              ],
            ),
          );
        }
        return const Center(child: CircularProgressIndicator());
      },
    );
  }

  Widget _kv(String k, String v) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 2),
        child: Text('$k: $v'),
      );

  String _statusLabel(S loc, String status) => switch (status) {
        'DRAFT' => loc.salesStatusDraft,
        'SUBMITTED' => loc.salesStatusSubmitted,
        'APPROVED' => loc.salesStatusApproved,
        'REJECTED' => loc.salesStatusRejected,
        'COMPLETED' => loc.salesStatusCompleted,
        'CANCELLED' => loc.salesStatusCancelled,
        _ => status,
      };
}
