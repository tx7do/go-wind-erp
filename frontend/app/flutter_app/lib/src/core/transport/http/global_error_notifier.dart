import 'package:flutter/foundation.dart' show ChangeNotifier;

/// 全局错误通知器。
///
/// 统一拦截器在遇到“非预期”错误（5xx / 网络失败 / 未知）时调用 [fire]，
/// 根 widget 监听并展示一条通用错误提示。鉴权类错误不上报（由会话失效
/// 流程处理：清除令牌 + 路由重定向至登录）；4xx 业务错误亦不上报
/// （由调用方 UI 自行处理，例如登录表单）。
class GlobalErrorNotifier extends ChangeNotifier {
  void fire() => notifyListeners();
}
