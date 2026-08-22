import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:go_wind_erp/generated/l10n.dart';
import 'package:go_wind_erp/src/features/wms/presentation/wms_cubit.dart';
import 'package:go_wind_erp/src/features/wms/presentation/wms_state.dart';

/// WMS 扫码作业页。
///
/// 流程：选择仓库 → 输入/扫描 productCode 查库存量 → 发起内部调拨拣货单
/// （create→confirm→validate）→ 展示近期拣货单。入库拣货单不再由客户端
/// 创建——服务层在采购单审批通过后自动生成。
class WmsPage extends StatefulWidget {
  const WmsPage({super.key});

  @override
  State<WmsPage> createState() => _WmsPageState();
}

class _WmsPageState extends State<WmsPage> {
  final _skuController = TextEditingController();

  @override
  void dispose() {
    _skuController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(loc.navWms)),
      body: BlocConsumer<WmsCubit, WmsState>(
        listener: (context, state) {
          if (state is WmsReady && state.message != null) {
            final text = switch (state.message) {
              'transferSuccess' => loc.transferSuccess,
              'transferFailed' => loc.transferFailed,
              'negativeStock' => loc.negativeStock,
              'lookupFailed' => loc.lookupFailed,
              'sameWarehouse' => loc.sameWarehouse,
              _ => state.message!,
            };
            ScaffoldMessenger.of(context)
              ..hideCurrentSnackBar()
              ..showSnackBar(SnackBar(content: Text(text)));
            context.read<WmsCubit>().clearMessage();
          }
        },
        builder: (context, state) {
          if (state is WmsReady) {
            return _buildReady(context, loc, state);
          }
          if (state is WmsLoadFailure) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(loc.loadFailed),
                  const SizedBox(height: 12),
                  FilledButton(
                    onPressed: () => context.read<WmsCubit>().load(),
                    child: Text(loc.retry),
                  ),
                ],
              ),
            );
          }
          return const Center(child: CircularProgressIndicator());
        },
      ),
    );
  }

  Widget _buildReady(BuildContext context, S loc, WmsReady state) {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _buildWarehousePicker(loc, state),
          const SizedBox(height: 12),
          _buildSkuLookup(loc, state),
          const SizedBox(height: 12),
          if (state.lookupMiss)
            Text(loc.lookupMiss, style: const TextStyle(color: Colors.orange)),
          if (state.inventory != null) ...[
            _buildInventoryCard(loc, state),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: state.submitting
                  ? null
                  : () => _showTransferDialog(context, loc, state),
              icon: const Icon(Icons.swap_horiz),
              label: Text(loc.transferAction),
            ),
          ],
          if (state.pickings.isNotEmpty) ...[
            const SizedBox(height: 12),
            _buildPickings(loc, state),
          ],
        ],
      ),
    );
  }

  Widget _buildWarehousePicker(S loc, WmsReady state) {
    if (state.warehouses.isEmpty) {
      return Text(loc.noWarehouse);
    }
    return DropdownButtonFormField<String>(
      initialValue: state.selectedWarehouseCode,
      decoration: InputDecoration(
        labelText: loc.selectWarehouse,
        border: const OutlineInputBorder(),
        isDense: true,
      ),
      items: [
        for (final w in state.warehouses)
          DropdownMenuItem(value: w.code, child: Text('${w.code} — ${w.name}')),
      ],
      onChanged: (code) {
        if (code != null) {
          _skuController.clear();
          context.read<WmsCubit>().selectWarehouse(code);
        }
      },
    );
  }

  Widget _buildSkuLookup(S loc, WmsReady state) {
    return Row(
      children: [
        Expanded(
          child: TextField(
            controller: _skuController,
            decoration: InputDecoration(
              labelText: loc.skuCodeHint,
              border: const OutlineInputBorder(),
              isDense: true,
            ),
            onSubmitted: (_) => _lookup(context, state),
          ),
        ),
        const SizedBox(width: 8),
        FilledButton(
          onPressed: state.lookingUp ? null : () => _lookup(context, state),
          child: state.lookingUp
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Text(loc.lookup),
        ),
      ],
    );
  }

  void _lookup(BuildContext context, WmsReady state) {
    if (state.selectedWarehouseCode == null) {
      _showSnackBar(context, S.of(context).pickWarehouseFirst);
      return;
    }
    context.read<WmsCubit>().lookupInventory(_skuController.text);
  }

  Widget _buildInventoryCard(S loc, WmsReady state) {
    final inv = state.inventory!;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              inv.productCode,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 4),
            Text('${loc.inventoryQuantity}: ${inv.quantity}'),
          ],
        ),
      ),
    );
  }

  /// 内部调拨对话框：目的仓库下拉（排除当前仓）+ 数量。
  ///
  /// 提交后由 [WmsCubit.submitInternalTransfer] 走 create→confirm→validate
  /// 创建 INTERNAL 拣货单；source/dest location 由服务层按仓库推导落库。
  void _showTransferDialog(BuildContext context, S loc, WmsReady state) {
    final inv = state.inventory;
    if (inv == null) {
      _showSnackBar(context, loc.scanSkuFirst);
      return;
    }
    final candidates = state.warehouses
        .where((w) => w.code != state.selectedWarehouseCode)
        .toList();
    if (candidates.isEmpty) {
      _showSnackBar(context, loc.sameWarehouse);
      return;
    }

    final qtyController = TextEditingController();
    var toWarehouse = candidates.first.code;

    showDialog<void>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (dialogContext, setDialogState) => AlertDialog(
          title: Text(loc.transferAction),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<String>(
                initialValue: toWarehouse,
                decoration: InputDecoration(
                  labelText: loc.transferToWarehouse,
                  border: const OutlineInputBorder(),
                  isDense: true,
                ),
                items: [
                  for (final w in candidates)
                    DropdownMenuItem(
                      value: w.code,
                      child: Text('${w.code} — ${w.name}'),
                    ),
                ],
                onChanged: (code) {
                  if (code != null) setDialogState(() => toWarehouse = code);
                },
              ),
              const SizedBox(height: 12),
              TextField(
                controller: qtyController,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(
                  labelText: loc.quantityLabel,
                  border: const OutlineInputBorder(),
                  isDense: true,
                ),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: Text(loc.cancel),
            ),
            FilledButton(
              onPressed: () {
                final quantity = int.tryParse(qtyController.text.trim());
                if (quantity == null || quantity <= 0) {
                  _showSnackBar(dialogContext, loc.quantityInvalid);
                  return;
                }
                Navigator.of(dialogContext).pop();
                context.read<WmsCubit>().submitInternalTransfer(
                      toWarehouseCode: toWarehouse,
                      quantity: quantity,
                    );
              },
              child: Text(loc.confirm),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildPickings(S loc, WmsReady state) {
    return Card(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: Text(
              loc.recentMovements,
              style: Theme.of(context).textTheme.titleMedium,
            ),
          ),
          for (final p in state.pickings)
            ListTile(
              dense: true,
              leading: const Icon(Icons.swap_horiz, color: Colors.grey),
              title: Text(p.pickingNumber),
              subtitle: Text('${p.pickingType} · ${p.derivedState}'),
            ),
        ],
      ),
    );
  }

  void _showSnackBar(BuildContext context, String text) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(text)));
  }
}
