import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:get_it/get_it.dart';

import 'package:go_wind_erp/generated/l10n.dart';
import 'package:go_wind_erp/src/features/sales/domain/sales_models.dart';
import 'package:go_wind_erp/src/features/sales/domain/sales_repository.dart';
import 'package:go_wind_erp/src/features/sales/presentation/sales_cubit.dart';

/// 销售单详情页（只读单头 + 明细履约进度 + 销售退货入口）。
///
/// 退货按钮按明细已履约数显隐（服务端仍有状态门与防超退守卫）；
/// 退货成功后提示去拣货单列表执行确认+校验。
class SalesOrderDetailPage extends StatelessWidget {
  final int orderId;

  const SalesOrderDetailPage({super.key, required this.orderId});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return BlocProvider(
      create: (_) =>
          SalesDetailCubit(GetIt.instance<SalesRepository>())..load(orderId),
      child: Scaffold(
        appBar: AppBar(title: Text(loc.salesDetailTitle)),
        body: const _DetailBody(),
      ),
    );
  }
}

class _DetailBody extends StatelessWidget {
  const _DetailBody();

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return BlocConsumer<SalesDetailCubit, SalesDetailState>(
      listener: (context, state) {
        if (state is SalesDetailReady && state.message != null) {
          final msg = switch (state.message) {
            'returnCreated' => loc.salesReturnCreated,
            'returnFailed' => loc.salesReturnFailed,
            _ => loc.loadFailed,
          };
          ScaffoldMessenger.of(context)
            ..hideCurrentSnackBar()
            ..showSnackBar(SnackBar(content: Text(msg)));
          context.read<SalesDetailCubit>().consumeMessage();
        }
      },
      builder: (context, state) {
        if (state is SalesDetailReady) {
          return _DetailContent(order: state.order, returning: state.returning);
        }
        return const Center(child: CircularProgressIndicator());
      },
    );
  }
}

class _DetailContent extends StatelessWidget {
  final SalesOrderInfo order;
  final bool returning;

  const _DetailContent({required this.order, required this.returning});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
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
                  order.soNumber,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(height: 8),
                _kv(loc.salesCustomer, order.customerCode),
                _kv(loc.salesWarehouse, order.warehouseCode),
                _kv(
                  loc.salesTotal,
                  '¥${(order.totalAmount / 100).toStringAsFixed(2)}',
                ),
                _kv(loc.salesStatus, order.status),
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
                '${loc.salesFulfilled}: ${item.fulfilledQuantity}',
              ),
              trailing: order.returnable && item.returnable > 0
                  ? TextButton(
                      onPressed: returning
                          ? null
                          : () => _confirmReturn(context, loc, item),
                      child: Text(loc.salesReturn),
                    )
                  : null,
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

  Widget _kv(String k, String v) =>
      Padding(padding: const EdgeInsets.symmetric(vertical: 2), child: Text('$k: $v'));

  void _confirmReturn(
    BuildContext context,
    S loc,
    SalesOrderItemInfo item,
  ) {
    final controller = TextEditingController(text: '${item.returnable}');
    showDialog<void>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text('${loc.salesReturn} · ${item.skuCode}'),
        content: TextField(
          controller: controller,
          keyboardType: TextInputType.number,
          decoration: InputDecoration(
            labelText:
                '${loc.salesReturnQuantity} (≤ ${item.returnable})',
            border: const OutlineInputBorder(),
            isDense: true,
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: Text(loc.cancel),
          ),
          FilledButton(
            onPressed: () {
              final qty = int.tryParse(controller.text.trim()) ?? 0;
              Navigator.of(dialogContext).pop();
              if (qty <= 0 || qty > item.returnable) {
                ScaffoldMessenger.of(context)
                  ..hideCurrentSnackBar()
                  ..showSnackBar(
                    SnackBar(content: Text(loc.salesReturnQtyInvalid)),
                  );
                return;
              }
              context.read<SalesDetailCubit>().submitReturn(
                    SalesReturnDraft(
                      salesOrderId: order.id,
                      salesOrderItemId: item.id,
                      quantity: qty,
                    ),
                  );
            },
            child: Text(loc.confirm),
          ),
        ],
      ),
    );
  }
}
