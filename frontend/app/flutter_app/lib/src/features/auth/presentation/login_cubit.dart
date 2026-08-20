import 'package:bloc/bloc.dart';

import 'package:go_wind_erp/src/features/auth/domain/auth_failure.dart';
import 'package:go_wind_erp/src/features/auth/domain/auth_repository.dart';
import 'package:go_wind_erp/src/features/auth/domain/login_credentials.dart';
import 'package:go_wind_erp/src/features/auth/presentation/login_state.dart';

/// 登录视图模型。
///
/// 仅依赖 [AuthRepository] 抽象。提交后：
/// - 成功：会话缓存已在仓储内更新，登录状态通知器随之触发，路由将重定向
///   至 `/`（home shell），当前页面被卸载，故无需 emit 成功状态。
/// - 失败：emit [LoginFailure] 携带 [AuthFailure]，UI 据此展示提示。
class LoginCubit extends Cubit<LoginState> {
  final AuthRepository _repository;

  LoginCubit(this._repository) : super(const LoginInitial());

  Future<void> login(LoginCredentials credentials) async {
    emit(const LoginLoading());
    try {
      await _repository.login(credentials);
      // 成功：见类注释。
    } on AuthFailure catch (e) {
      if (!isClosed) emit(LoginFailure(e));
    }
  }
}
