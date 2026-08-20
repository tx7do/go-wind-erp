import 'package:equatable/equatable.dart';
import 'package:meta/meta.dart';

import 'package:go_wind_erp/src/features/auth/domain/auth_failure.dart';

/// 登录视图状态。
@immutable
sealed class LoginState extends Equatable {
  const LoginState();
}

final class LoginInitial extends LoginState {
  const LoginInitial();

  @override
  List<Object> get props => [];
}

final class LoginLoading extends LoginState {
  const LoginLoading();

  @override
  List<Object> get props => [];
}

final class LoginFailure extends LoginState {
  final AuthFailure failure;
  const LoginFailure(this.failure);

  @override
  List<Object> get props => [failure];
}
