import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:go_wind_erp/generated/l10n.dart';
import 'package:go_wind_erp/src/features/wms/domain/wms_models.dart';
import 'package:go_wind_erp/src/features/wms/presentation/wms_cubit.dart';
import 'package:go_wind_erp/src/features/wms/presentation/wms_state.dart';

/// WMS 扫码作业页。
///
/// 流程：选择仓库 → 输入/扫描 SKU 查库存 → 选择方向（入库/出库）与数量
/// 提交流水 → 展示近期流水。提交成功后自动刷新库存与流水。
class WmsPage extends StatefulWidget {
  const WmsPage({super.key});

  @override
  State<WmsPage> createState() => _WmsPageState();
}

class _WmsPageState extends State<WmsPage> {
  final _skuController = TextEditingController();
  final _quantityController = TextEditingController();
  final _remarkController = TextEditingController();
  MovementKind _kind = MovementKind.inbound;

  @override
  void dispose() {
    _skuController.dispose();
    _quantityController.dispose();
    _remarkController.dispose();
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
              'submitSuccess' => loc.submitSuccess,
              'submitFailed' => loc.submitFailed,
              'negativeStock' => loc.negativeStock,
              'lookupFailed' => loc.lookupFailed,
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
            _buildMovementForm(loc, state),
          ],
          if (state.movements.isNotEmpty) ...[
            const SizedBox(height: 12),
            _buildMovements(loc, state),
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
          _quantityController.clear();
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
            Row(
              children: [
                Text(
                  inv.skuCode,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(width: 8),
                _statusChip(loc, inv.status),
              ],
            ),
            const SizedBox(height: 4),
            Text('${loc.inventoryQuantity}: ${inv.quantity}'),
          ],
        ),
      ),
    );
  }

  Widget _statusChip(S loc, String status) {
    final (label, color) = switch (status) {
      'AVAILABLE' => (loc.statusAvailable, Colors.green),
      'LOCKED' => (loc.statusLocked, Colors.orange),
      'QUARANTINED' => (loc.statusQuarantined, Colors.red),
      _ => (status, Colors.grey),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(label, style: TextStyle(color: color, fontSize: 12)),
    );
  }

  Widget _buildMovementForm(S loc, WmsReady state) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            SegmentedButton<MovementKind>(
              segments: [
                ButtonSegment(
                  value: MovementKind.inbound,
                  label: Text(loc.inbound),
                  icon: const Icon(Icons.south_west),
                ),
                ButtonSegment(
                  value: MovementKind.outbound,
                  label: Text(loc.outbound),
                  icon: const Icon(Icons.north_east),
                ),
              ],
              selected: {_kind},
              onSelectionChanged: (set) =>
                  setState(() => _kind = set.first),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _quantityController,
              keyboardType: TextInputType.number,
              decoration: InputDecoration(
                labelText: loc.quantityLabel,
                border: const OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _remarkController,
              decoration: InputDecoration(
                labelText: loc.remarkLabel,
                border: const OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 12),
            FilledButton(
              onPressed: state.submitting ? null : () => _submit(context, state),
              child: state.submitting
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(loc.submitMovement),
            ),
          ],
        ),
      ),
    );
  }

  void _submit(BuildContext context, WmsReady state) {
    final inv = state.inventory;
    if (inv == null) {
      _showSnackBar(context, S.of(context).scanSkuFirst);
      return;
    }
    final quantity = int.tryParse(_quantityController.text.trim());
    if (quantity == null || quantity <= 0) {
      _showSnackBar(context, S.of(context).quantityInvalid);
      return;
    }
    context.read<WmsCubit>().submitMovement(
          kind: _kind,
          quantity: quantity,
          remark: _remarkController.text.trim(),
        );
    _quantityController.clear();
    _remarkController.clear();
  }

  Widget _buildMovements(S loc, WmsReady state) {
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
          for (final m in state.movements)
            ListTile(
              dense: true,
              leading: Icon(
                m.delta >= 0 ? Icons.south_west : Icons.north_east,
                color: m.delta >= 0 ? Colors.green : Colors.red,
              ),
              title: Text('${_movementLabel(loc, m.movementType)}  ${m.delta >= 0 ? '+' : ''}${m.delta}'),
              subtitle: Text('${m.quantityBefore} → ${m.quantityAfter}'),
            ),
        ],
      ),
    );
  }

  String _movementLabel(S loc, String type) => switch (type) {
        'INBOUND' => loc.inbound,
        'OUTBOUND' => loc.outbound,
        'TRANSFER' => 'TRANSFER',
        'ADJUSTMENT' => 'ADJUSTMENT',
        _ => type,
      };

  void _showSnackBar(BuildContext context, String text) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(text)));
  }
}
