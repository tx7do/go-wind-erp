// DO NOT EDIT. This is code generated via package:intl/generate_localized.dart
// This is a library that provides messages for a zh_CN locale. All the
// messages from the main program should be duplicated here with the same
// function name.

// Ignore issues from commonly used lints in this file.
// ignore_for_file:unnecessary_brace_in_string_interps, unnecessary_new
// ignore_for_file:prefer_single_quotes,comment_references, directives_ordering
// ignore_for_file:annotate_overrides,prefer_generic_function_type_aliases
// ignore_for_file:unused_import, file_names, avoid_escaping_inner_quotes
// ignore_for_file:unnecessary_string_interpolations, unnecessary_string_escapes

import 'package:intl/intl.dart';
import 'package:intl/message_lookup_by_library.dart';

final messages = new MessageLookup();

typedef String MessageIfAbsent(String messageStr, List<dynamic> args);

class MessageLookup extends MessageLookupByLibrary {
  String get localeName => 'zh_CN';

  final messages = _notInlinedMessages(_notInlinedMessages);
  static Map<String, Function> _notInlinedMessages(_) => <String, Function>{
    "appName": MessageLookupByLibrary.simpleMessage("GoWind ERP"),
    "appearance": MessageLookupByLibrary.simpleMessage("外观设置"),
    "approvalActionRejected": MessageLookupByLibrary.simpleMessage(
      "操作被拒绝：状态已变更或无权限",
    ),
    "approvalApprove": MessageLookupByLibrary.simpleMessage("通过"),
    "approvalCancel": MessageLookupByLibrary.simpleMessage("撤销"),
    "approvalCancelConfirm": MessageLookupByLibrary.simpleMessage(
      "确定撤销该审批请求吗？",
    ),
    "approvalComment": MessageLookupByLibrary.simpleMessage("审批意见"),
    "approvalEmpty": MessageLookupByLibrary.simpleMessage("暂无审批请求"),
    "approvalFilterAll": MessageLookupByLibrary.simpleMessage("全部"),
    "approvalReject": MessageLookupByLibrary.simpleMessage("驳回"),
    "approvalStatusApproved": MessageLookupByLibrary.simpleMessage("已通过"),
    "approvalStatusCancelled": MessageLookupByLibrary.simpleMessage("已撤销"),
    "approvalStatusPending": MessageLookupByLibrary.simpleMessage("待审批"),
    "approvalStatusRejected": MessageLookupByLibrary.simpleMessage("已驳回"),
    "backToHome": MessageLookupByLibrary.simpleMessage("返回首页"),
    "cancel": MessageLookupByLibrary.simpleMessage("取消"),
    "comingSoon": MessageLookupByLibrary.simpleMessage("模块开发中"),
    "confirm": MessageLookupByLibrary.simpleMessage("确定"),
    "dark": MessageLookupByLibrary.simpleMessage("深色"),
    "darkMode": MessageLookupByLibrary.simpleMessage("深色模式"),
    "errorOccurred": MessageLookupByLibrary.simpleMessage("发生错误！"),
    "followSystem": MessageLookupByLibrary.simpleMessage("跟随系统"),
    "inbound": MessageLookupByLibrary.simpleMessage("入库"),
    "inventoryQuantity": MessageLookupByLibrary.simpleMessage("当前库存"),
    "language": MessageLookupByLibrary.simpleMessage("语言"),
    "light": MessageLookupByLibrary.simpleMessage("浅色"),
    "loadFailed": MessageLookupByLibrary.simpleMessage("加载失败"),
    "login": MessageLookupByLibrary.simpleMessage("登录"),
    "loginButton": MessageLookupByLibrary.simpleMessage("登录"),
    "loginFailed": MessageLookupByLibrary.simpleMessage("登录失败，请检查用户名和密码"),
    "loginForMore": MessageLookupByLibrary.simpleMessage("登录后享受更多功能"),
    "loginSuccess": MessageLookupByLibrary.simpleMessage("登录成功"),
    "logout": MessageLookupByLibrary.simpleMessage("退出登录"),
    "lookup": MessageLookupByLibrary.simpleMessage("查询"),
    "lookupFailed": MessageLookupByLibrary.simpleMessage("查询失败"),
    "lookupMiss": MessageLookupByLibrary.simpleMessage("未找到该 SKU 的库存记录"),
    "lowStockEmpty": MessageLookupByLibrary.simpleMessage("暂无低库存项"),
    "lowStockTitle": MessageLookupByLibrary.simpleMessage("低库存清单"),
    "metricMovements": MessageLookupByLibrary.simpleMessage("流水数"),
    "metricSkus": MessageLookupByLibrary.simpleMessage("在库 SKU"),
    "metricTotalQuantity": MessageLookupByLibrary.simpleMessage("库存总量"),
    "metricWarehouses": MessageLookupByLibrary.simpleMessage("仓库数"),
    "navApproval": MessageLookupByLibrary.simpleMessage("审批"),
    "navDashboard": MessageLookupByLibrary.simpleMessage("看板"),
    "navWms": MessageLookupByLibrary.simpleMessage("仓储"),
    "negativeStock": MessageLookupByLibrary.simpleMessage("出库数量不能超过当前库存"),
    "noWarehouse": MessageLookupByLibrary.simpleMessage("暂无可用仓库"),
    "outbound": MessageLookupByLibrary.simpleMessage("出库"),
    "pageNotFound": MessageLookupByLibrary.simpleMessage("页面未找到"),
    "pageNotFoundDesc": MessageLookupByLibrary.simpleMessage(
      "抱歉，您访问的页面不存在或已被移动。",
    ),
    "password": MessageLookupByLibrary.simpleMessage("密码"),
    "passwordHint": MessageLookupByLibrary.simpleMessage("请输入密码"),
    "pickWarehouseFirst": MessageLookupByLibrary.simpleMessage("请先选择仓库"),
    "quantityInvalid": MessageLookupByLibrary.simpleMessage("请输入正整数数量"),
    "quantityLabel": MessageLookupByLibrary.simpleMessage("数量"),
    "recentMovements": MessageLookupByLibrary.simpleMessage("近期流水"),
    "remarkLabel": MessageLookupByLibrary.simpleMessage("备注"),
    "retry": MessageLookupByLibrary.simpleMessage("重试"),
    "reverseAction": MessageLookupByLibrary.simpleMessage("冲正"),
    "reverseFailed": MessageLookupByLibrary.simpleMessage("冲正失败"),
    "reverseReason": MessageLookupByLibrary.simpleMessage("冲正原因"),
    "reverseReasonRequired": MessageLookupByLibrary.simpleMessage("请填写冲正原因"),
    "reverseSuccess": MessageLookupByLibrary.simpleMessage("冲正成功"),
    "sameWarehouse": MessageLookupByLibrary.simpleMessage("源仓库与目的仓库不能相同"),
    "scanSkuFirst": MessageLookupByLibrary.simpleMessage("请先查询 SKU 库存"),
    "selectWarehouse": MessageLookupByLibrary.simpleMessage("选择仓库"),
    "skuCodeHint": MessageLookupByLibrary.simpleMessage("输入/扫描 SKU"),
    "statusAvailable": MessageLookupByLibrary.simpleMessage("可用"),
    "statusLocked": MessageLookupByLibrary.simpleMessage("锁定"),
    "statusQuarantined": MessageLookupByLibrary.simpleMessage("隔离"),
    "submitFailed": MessageLookupByLibrary.simpleMessage("提交失败"),
    "submitMovement": MessageLookupByLibrary.simpleMessage("提交流水"),
    "submitSuccess": MessageLookupByLibrary.simpleMessage("提交成功"),
    "tenantCode": MessageLookupByLibrary.simpleMessage("租户编号"),
    "tenantCodeHint": MessageLookupByLibrary.simpleMessage("留空表示平台登录"),
    "themeColor": MessageLookupByLibrary.simpleMessage("主题色"),
    "transferAction": MessageLookupByLibrary.simpleMessage("调拨"),
    "transferFailed": MessageLookupByLibrary.simpleMessage("调拨失败"),
    "transferSuccess": MessageLookupByLibrary.simpleMessage("调拨成功"),
    "transferToWarehouse": MessageLookupByLibrary.simpleMessage("目的仓库"),
    "username": MessageLookupByLibrary.simpleMessage("用户名"),
    "usernameHint": MessageLookupByLibrary.simpleMessage("请输入用户名"),
  };
}
