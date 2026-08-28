// GENERATED CODE - DO NOT MODIFY BY HAND
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'intl/messages_all.dart';

// **************************************************************************
// Generator: Flutter Intl IDE plugin
// Made by Localizely
// **************************************************************************

// ignore_for_file: non_constant_identifier_names, lines_longer_than_80_chars
// ignore_for_file: join_return_with_assignment, prefer_final_in_for_each
// ignore_for_file: avoid_redundant_argument_values, avoid_escaping_inner_quotes

class S {
  S();

  static S? _current;

  static S get current {
    assert(
      _current != null,
      'No instance of S was loaded. Try to initialize the S delegate before accessing S.current.',
    );
    return _current!;
  }

  static const AppLocalizationDelegate delegate = AppLocalizationDelegate();

  static Future<S> load(Locale locale) {
    final name = (locale.countryCode?.isEmpty ?? false)
        ? locale.languageCode
        : locale.toString();
    final localeName = Intl.canonicalizedLocale(name);
    return initializeMessages(localeName).then((_) {
      Intl.defaultLocale = localeName;
      final instance = S();
      S._current = instance;

      return instance;
    });
  }

  static S of(BuildContext context) {
    final instance = S.maybeOf(context);
    assert(
      instance != null,
      'No instance of S present in the widget tree. Did you add S.delegate in localizationsDelegates?',
    );
    return instance!;
  }

  static S? maybeOf(BuildContext context) {
    return Localizations.of<S>(context, S);
  }

  /// `GoWind ERP`
  String get appName {
    return Intl.message('GoWind ERP', name: 'appName', desc: '', args: []);
  }

  /// `登录`
  String get login {
    return Intl.message('登录', name: 'login', desc: '', args: []);
  }

  /// `登录后享受更多功能`
  String get loginForMore {
    return Intl.message('登录后享受更多功能', name: 'loginForMore', desc: '', args: []);
  }

  /// `外观设置`
  String get appearance {
    return Intl.message('外观设置', name: 'appearance', desc: '', args: []);
  }

  /// `语言`
  String get language {
    return Intl.message('语言', name: 'language', desc: '', args: []);
  }

  /// `主题色`
  String get themeColor {
    return Intl.message('主题色', name: 'themeColor', desc: '', args: []);
  }

  /// `深色模式`
  String get darkMode {
    return Intl.message('深色模式', name: 'darkMode', desc: '', args: []);
  }

  /// `浅色`
  String get light {
    return Intl.message('浅色', name: 'light', desc: '', args: []);
  }

  /// `跟随系统`
  String get followSystem {
    return Intl.message('跟随系统', name: 'followSystem', desc: '', args: []);
  }

  /// `深色`
  String get dark {
    return Intl.message('深色', name: 'dark', desc: '', args: []);
  }

  /// `发生错误！`
  String get errorOccurred {
    return Intl.message('发生错误！', name: 'errorOccurred', desc: '', args: []);
  }

  /// `页面未找到`
  String get pageNotFound {
    return Intl.message('页面未找到', name: 'pageNotFound', desc: '', args: []);
  }

  /// `抱歉，您访问的页面不存在或已被移动。`
  String get pageNotFoundDesc {
    return Intl.message(
      '抱歉，您访问的页面不存在或已被移动。',
      name: 'pageNotFoundDesc',
      desc: '',
      args: [],
    );
  }

  /// `返回首页`
  String get backToHome {
    return Intl.message('返回首页', name: 'backToHome', desc: '', args: []);
  }

  /// `用户名`
  String get username {
    return Intl.message('用户名', name: 'username', desc: '', args: []);
  }

  /// `密码`
  String get password {
    return Intl.message('密码', name: 'password', desc: '', args: []);
  }

  /// `请输入用户名`
  String get usernameHint {
    return Intl.message('请输入用户名', name: 'usernameHint', desc: '', args: []);
  }

  /// `请输入密码`
  String get passwordHint {
    return Intl.message('请输入密码', name: 'passwordHint', desc: '', args: []);
  }

  /// `登录`
  String get loginButton {
    return Intl.message('登录', name: 'loginButton', desc: '', args: []);
  }

  /// `登录成功`
  String get loginSuccess {
    return Intl.message('登录成功', name: 'loginSuccess', desc: '', args: []);
  }

  /// `登录失败，请检查用户名和密码`
  String get loginFailed {
    return Intl.message(
      '登录失败，请检查用户名和密码',
      name: 'loginFailed',
      desc: '',
      args: [],
    );
  }

  /// `退出登录`
  String get logout {
    return Intl.message('退出登录', name: 'logout', desc: '', args: []);
  }

  /// `取消`
  String get cancel {
    return Intl.message('取消', name: 'cancel', desc: '', args: []);
  }

  /// `确定`
  String get confirm {
    return Intl.message('确定', name: 'confirm', desc: '', args: []);
  }

  /// `租户编号`
  String get tenantCode {
    return Intl.message('租户编号', name: 'tenantCode', desc: '', args: []);
  }

  /// `留空表示平台登录`
  String get tenantCodeHint {
    return Intl.message('留空表示平台登录', name: 'tenantCodeHint', desc: '', args: []);
  }

  /// `看板`
  String get navDashboard {
    return Intl.message('看板', name: 'navDashboard', desc: '', args: []);
  }

  /// `审批`
  String get navApproval {
    return Intl.message('审批', name: 'navApproval', desc: '', args: []);
  }

  /// `仓储`
  String get navWms {
    return Intl.message('仓储', name: 'navWms', desc: '', args: []);
  }

  /// `模块开发中`
  String get comingSoon {
    return Intl.message('模块开发中', name: 'comingSoon', desc: '', args: []);
  }

  /// `加载失败`
  String get loadFailed {
    return Intl.message('加载失败', name: 'loadFailed', desc: '', args: []);
  }

  /// `重试`
  String get retry {
    return Intl.message('重试', name: 'retry', desc: '', args: []);
  }

  /// `暂无可用仓库`
  String get noWarehouse {
    return Intl.message('暂无可用仓库', name: 'noWarehouse', desc: '', args: []);
  }

  /// `选择仓库`
  String get selectWarehouse {
    return Intl.message('选择仓库', name: 'selectWarehouse', desc: '', args: []);
  }

  /// `输入/扫描 SKU`
  String get skuCodeHint {
    return Intl.message('输入/扫描 SKU', name: 'skuCodeHint', desc: '', args: []);
  }

  /// `查询`
  String get lookup {
    return Intl.message('查询', name: 'lookup', desc: '', args: []);
  }

  /// `未找到该 SKU 的库存记录`
  String get lookupMiss {
    return Intl.message(
      '未找到该 SKU 的库存记录',
      name: 'lookupMiss',
      desc: '',
      args: [],
    );
  }

  /// `查询失败`
  String get lookupFailed {
    return Intl.message('查询失败', name: 'lookupFailed', desc: '', args: []);
  }

  /// `可用`
  String get statusAvailable {
    return Intl.message('可用', name: 'statusAvailable', desc: '', args: []);
  }

  /// `锁定`
  String get statusLocked {
    return Intl.message('锁定', name: 'statusLocked', desc: '', args: []);
  }

  /// `隔离`
  String get statusQuarantined {
    return Intl.message('隔离', name: 'statusQuarantined', desc: '', args: []);
  }

  /// `当前库存`
  String get inventoryQuantity {
    return Intl.message('当前库存', name: 'inventoryQuantity', desc: '', args: []);
  }

  /// `入库`
  String get inbound {
    return Intl.message('入库', name: 'inbound', desc: '', args: []);
  }

  /// `出库`
  String get outbound {
    return Intl.message('出库', name: 'outbound', desc: '', args: []);
  }

  /// `数量`
  String get quantityLabel {
    return Intl.message('数量', name: 'quantityLabel', desc: '', args: []);
  }

  /// `备注`
  String get remarkLabel {
    return Intl.message('备注', name: 'remarkLabel', desc: '', args: []);
  }

  /// `提交流水`
  String get submitMovement {
    return Intl.message('提交流水', name: 'submitMovement', desc: '', args: []);
  }

  /// `提交成功`
  String get submitSuccess {
    return Intl.message('提交成功', name: 'submitSuccess', desc: '', args: []);
  }

  /// `提交失败`
  String get submitFailed {
    return Intl.message('提交失败', name: 'submitFailed', desc: '', args: []);
  }

  /// `调拨`
  String get transferAction {
    return Intl.message('调拨', name: 'transferAction', desc: '', args: []);
  }

  /// `目的仓库`
  String get transferToWarehouse {
    return Intl.message(
      '目的仓库',
      name: 'transferToWarehouse',
      desc: '',
      args: [],
    );
  }

  /// `调拨成功`
  String get transferSuccess {
    return Intl.message('调拨成功', name: 'transferSuccess', desc: '', args: []);
  }

  /// `调拨失败`
  String get transferFailed {
    return Intl.message('调拨失败', name: 'transferFailed', desc: '', args: []);
  }

  /// `冲正`
  String get reverseAction {
    return Intl.message('冲正', name: 'reverseAction', desc: '', args: []);
  }

  /// `冲正原因`
  String get reverseReason {
    return Intl.message('冲正原因', name: 'reverseReason', desc: '', args: []);
  }

  /// `请填写冲正原因`
  String get reverseReasonRequired {
    return Intl.message(
      '请填写冲正原因',
      name: 'reverseReasonRequired',
      desc: '',
      args: [],
    );
  }

  /// `冲正成功`
  String get reverseSuccess {
    return Intl.message('冲正成功', name: 'reverseSuccess', desc: '', args: []);
  }

  /// `冲正失败`
  String get reverseFailed {
    return Intl.message('冲正失败', name: 'reverseFailed', desc: '', args: []);
  }

  /// `源仓库与目的仓库不能相同`
  String get sameWarehouse {
    return Intl.message(
      '源仓库与目的仓库不能相同',
      name: 'sameWarehouse',
      desc: '',
      args: [],
    );
  }

  /// `出库数量不能超过当前库存`
  String get negativeStock {
    return Intl.message(
      '出库数量不能超过当前库存',
      name: 'negativeStock',
      desc: '',
      args: [],
    );
  }

  /// `请先查询 SKU 库存`
  String get scanSkuFirst {
    return Intl.message(
      '请先查询 SKU 库存',
      name: 'scanSkuFirst',
      desc: '',
      args: [],
    );
  }

  /// `请先选择仓库`
  String get pickWarehouseFirst {
    return Intl.message(
      '请先选择仓库',
      name: 'pickWarehouseFirst',
      desc: '',
      args: [],
    );
  }

  /// `请输入正整数数量`
  String get quantityInvalid {
    return Intl.message(
      '请输入正整数数量',
      name: 'quantityInvalid',
      desc: '',
      args: [],
    );
  }

  /// `近期流水`
  String get recentMovements {
    return Intl.message('近期流水', name: 'recentMovements', desc: '', args: []);
  }

  /// `仓库数`
  String get metricWarehouses {
    return Intl.message('仓库数', name: 'metricWarehouses', desc: '', args: []);
  }

  /// `在库 SKU`
  String get metricSkus {
    return Intl.message('在库 SKU', name: 'metricSkus', desc: '', args: []);
  }

  /// `库存总量`
  String get metricTotalQuantity {
    return Intl.message(
      '库存总量',
      name: 'metricTotalQuantity',
      desc: '',
      args: [],
    );
  }

  /// `流水数`
  String get metricMovements {
    return Intl.message('流水数', name: 'metricMovements', desc: '', args: []);
  }

  /// `低库存清单`
  String get lowStockTitle {
    return Intl.message('低库存清单', name: 'lowStockTitle', desc: '', args: []);
  }

  /// `暂无低库存项`
  String get lowStockEmpty {
    return Intl.message('暂无低库存项', name: 'lowStockEmpty', desc: '', args: []);
  }

  /// `暂无审批请求`
  String get approvalEmpty {
    return Intl.message('暂无审批请求', name: 'approvalEmpty', desc: '', args: []);
  }

  /// `全部`
  String get approvalFilterAll {
    return Intl.message('全部', name: 'approvalFilterAll', desc: '', args: []);
  }

  /// `待审批`
  String get approvalStatusPending {
    return Intl.message(
      '待审批',
      name: 'approvalStatusPending',
      desc: '',
      args: [],
    );
  }

  /// `已通过`
  String get approvalStatusApproved {
    return Intl.message(
      '已通过',
      name: 'approvalStatusApproved',
      desc: '',
      args: [],
    );
  }

  /// `已驳回`
  String get approvalStatusRejected {
    return Intl.message(
      '已驳回',
      name: 'approvalStatusRejected',
      desc: '',
      args: [],
    );
  }

  /// `已撤销`
  String get approvalStatusCancelled {
    return Intl.message(
      '已撤销',
      name: 'approvalStatusCancelled',
      desc: '',
      args: [],
    );
  }

  /// `通过`
  String get approvalApprove {
    return Intl.message('通过', name: 'approvalApprove', desc: '', args: []);
  }

  /// `驳回`
  String get approvalReject {
    return Intl.message('驳回', name: 'approvalReject', desc: '', args: []);
  }

  /// `撤销`
  String get approvalCancel {
    return Intl.message('撤销', name: 'approvalCancel', desc: '', args: []);
  }

  /// `确定撤销该审批请求吗？`
  String get approvalCancelConfirm {
    return Intl.message(
      '确定撤销该审批请求吗？',
      name: 'approvalCancelConfirm',
      desc: '',
      args: [],
    );
  }

  /// `审批意见`
  String get approvalComment {
    return Intl.message('审批意见', name: 'approvalComment', desc: '', args: []);
  }

  /// `操作被拒绝：状态已变更或无权限`
  String get approvalActionRejected {
    return Intl.message(
      '操作被拒绝：状态已变更或无权限',
      name: 'approvalActionRejected',
      desc: '',
      args: [],
    );
  }

  /// `入库`
  String get pickingTypeIncoming {
    return Intl.message('入库', name: 'pickingTypeIncoming', desc: '', args: []);
  }

  /// `内部调拨`
  String get pickingTypeInternal {
    return Intl.message(
      '内部调拨',
      name: 'pickingTypeInternal',
      desc: '',
      args: [],
    );
  }

  /// `草稿`
  String get pickingStateDraft {
    return Intl.message('草稿', name: 'pickingStateDraft', desc: '', args: []);
  }

  /// `已确认`
  String get pickingStateConfirmed {
    return Intl.message(
      '已确认',
      name: 'pickingStateConfirmed',
      desc: '',
      args: [],
    );
  }

  /// `已完成`
  String get pickingStateDone {
    return Intl.message('已完成', name: 'pickingStateDone', desc: '', args: []);
  }

  /// `已取消`
  String get pickingStateCancelled {
    return Intl.message(
      '已取消',
      name: 'pickingStateCancelled',
      desc: '',
      args: [],
    );
  }

  /// `销售`
  String get navSales {
    return Intl.message('销售', name: 'navSales', desc: '', args: []);
  }

  /// `暂无销售单`
  String get salesEmpty {
    return Intl.message('暂无销售单', name: 'salesEmpty', desc: '', args: []);
  }

  /// `客户`
  String get salesCustomer {
    return Intl.message('客户', name: 'salesCustomer', desc: '', args: []);
  }

  /// `仓库`
  String get salesWarehouse {
    return Intl.message('仓库', name: 'salesWarehouse', desc: '', args: []);
  }

  /// `总额`
  String get salesTotal {
    return Intl.message('总额', name: 'salesTotal', desc: '', args: []);
  }

  /// `状态`
  String get salesStatus {
    return Intl.message('状态', name: 'salesStatus', desc: '', args: []);
  }

  /// `备注`
  String get salesRemark {
    return Intl.message('备注', name: 'salesRemark', desc: '', args: []);
  }

  /// `明细`
  String get salesItems {
    return Intl.message('明细', name: 'salesItems', desc: '', args: []);
  }

  /// `数量`
  String get salesQuantity {
    return Intl.message('数量', name: 'salesQuantity', desc: '', args: []);
  }

  /// `已履约`
  String get salesFulfilled {
    return Intl.message('已履约', name: 'salesFulfilled', desc: '', args: []);
  }

  /// `销售单`
  String get salesDetailTitle {
    return Intl.message('销售单', name: 'salesDetailTitle', desc: '', args: []);
  }

  /// `全部`
  String get salesFilterAll {
    return Intl.message('全部', name: 'salesFilterAll', desc: '', args: []);
  }

  /// `草稿`
  String get salesStatusDraft {
    return Intl.message('草稿', name: 'salesStatusDraft', desc: '', args: []);
  }

  /// `待审批`
  String get salesStatusSubmitted {
    return Intl.message(
      '待审批',
      name: 'salesStatusSubmitted',
      desc: '',
      args: [],
    );
  }

  /// `已通过`
  String get salesStatusApproved {
    return Intl.message('已通过', name: 'salesStatusApproved', desc: '', args: []);
  }

  /// `已驳回`
  String get salesStatusRejected {
    return Intl.message('已驳回', name: 'salesStatusRejected', desc: '', args: []);
  }

  /// `已完结`
  String get salesStatusCompleted {
    return Intl.message(
      '已完结',
      name: 'salesStatusCompleted',
      desc: '',
      args: [],
    );
  }

  /// `已取消`
  String get salesStatusCancelled {
    return Intl.message(
      '已取消',
      name: 'salesStatusCancelled',
      desc: '',
      args: [],
    );
  }

  /// `退货`
  String get salesReturn {
    return Intl.message('退货', name: 'salesReturn', desc: '', args: []);
  }

  /// `退货数量`
  String get salesReturnQuantity {
    return Intl.message(
      '退货数量',
      name: 'salesReturnQuantity',
      desc: '',
      args: [],
    );
  }

  /// `退货数量须在 1..已履约 范围内`
  String get salesReturnQtyInvalid {
    return Intl.message(
      '退货数量须在 1..已履约 范围内',
      name: 'salesReturnQtyInvalid',
      desc: '',
      args: [],
    );
  }

  /// `退货拣货单已创建，请到拣货单列表执行确认与校验`
  String get salesReturnCreated {
    return Intl.message(
      '退货拣货单已创建，请到拣货单列表执行确认与校验',
      name: 'salesReturnCreated',
      desc: '',
      args: [],
    );
  }

  /// `退货单创建失败`
  String get salesReturnFailed {
    return Intl.message(
      '退货单创建失败',
      name: 'salesReturnFailed',
      desc: '',
      args: [],
    );
  }

  /// `盘点`
  String get stocktakeAction {
    return Intl.message('盘点', name: 'stocktakeAction', desc: '', args: []);
  }

  /// `差异数量`
  String get stocktakeDiffLabel {
    return Intl.message('差异数量', name: 'stocktakeDiffLabel', desc: '', args: []);
  }

  /// `正数=盘盈，负数=盘亏`
  String get stocktakeDiffHint {
    return Intl.message(
      '正数=盘盈，负数=盘亏',
      name: 'stocktakeDiffHint',
      desc: '',
      args: [],
    );
  }

  /// `差异数量不能为 0`
  String get stocktakeZeroDiff {
    return Intl.message(
      '差异数量不能为 0',
      name: 'stocktakeZeroDiff',
      desc: '',
      args: [],
    );
  }

  /// `盘点已生效`
  String get stocktakeSuccess {
    return Intl.message('盘点已生效', name: 'stocktakeSuccess', desc: '', args: []);
  }

  /// `盘点失败`
  String get stocktakeFailed {
    return Intl.message('盘点失败', name: 'stocktakeFailed', desc: '', args: []);
  }

  /// `SKU`
  String get skuLabel {
    return Intl.message('SKU', name: 'skuLabel', desc: '', args: []);
  }

  /// `采购单`
  String get poDetailTitle {
    return Intl.message('采购单', name: 'poDetailTitle', desc: '', args: []);
  }

  /// `供应商`
  String get poSupplier {
    return Intl.message('供应商', name: 'poSupplier', desc: '', args: []);
  }

  /// `已收`
  String get poReceived {
    return Intl.message('已收', name: 'poReceived', desc: '', args: []);
  }
}

class AppLocalizationDelegate extends LocalizationsDelegate<S> {
  const AppLocalizationDelegate();

  List<Locale> get supportedLocales {
    return const <Locale>[
      Locale.fromSubtags(languageCode: 'zh', countryCode: 'CN'),
      Locale.fromSubtags(languageCode: 'en', countryCode: 'US'),
    ];
  }

  @override
  bool isSupported(Locale locale) => _isSupported(locale);
  @override
  Future<S> load(Locale locale) => S.load(locale);
  @override
  bool shouldReload(AppLocalizationDelegate old) => false;

  bool _isSupported(Locale locale) {
    for (var supportedLocale in supportedLocales) {
      if (supportedLocale.languageCode == locale.languageCode) {
        return true;
      }
    }
    return false;
  }
}
