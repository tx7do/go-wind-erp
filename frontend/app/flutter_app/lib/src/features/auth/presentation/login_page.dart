import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_screenutil/flutter_screenutil.dart';

import 'package:go_wind_erp/generated/l10n.dart';
import 'package:go_wind_erp/src/core/utils/responsive_utils.dart';
import 'package:go_wind_erp/src/features/auth/domain/login_credentials.dart';
import 'package:go_wind_erp/src/features/auth/presentation/login_cubit.dart';
import 'package:go_wind_erp/src/features/auth/presentation/login_state.dart';

/// 登录页。
///
/// 对接后端 `/app/v1/login` 契约：租户编号（可空，空表示平台登录）、
/// 用户名、密码。密码在 data 层按后端约定做 AES-CBC 加密。
/// 该端点不经白名单豁免之外的验证码——故无验证码字段。
class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _tenantController = TextEditingController();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _obscurePassword = true;

  @override
  void dispose() {
    _tenantController.dispose();
    _usernameController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  void _submit() {
    if (!_formKey.currentState!.validate()) return;
    context.read<LoginCubit>().login(
          LoginCredentials(
            username: _usernameController.text.trim(),
            password: _passwordController.text,
            tenantCode: _tenantController.text.trim(),
          ),
        );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isMobile = ResponsiveUtils.isMobile(context);
    final loc = S.of(context);

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: EdgeInsets.symmetric(
              horizontal: isMobile ? 24.w : 48,
              vertical: isMobile ? 32.h : 48,
            ),
            child: Container(
              constraints: BoxConstraints(
                maxWidth: isMobile ? double.infinity : 420,
              ),
              decoration: BoxDecoration(
                color: theme.scaffoldBackgroundColor,
                borderRadius: BorderRadius.circular(isMobile ? 20.r : 24),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withAlpha((0.06 * 255).round()),
                    blurRadius: 32,
                    offset: const Offset(0, 8),
                  ),
                ],
              ),
              child: Padding(
                padding: EdgeInsets.symmetric(
                  horizontal: isMobile ? 28.w : 40,
                  vertical: isMobile ? 32.h : 48,
                ),
                child: Form(
                  key: _formKey,
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Container(
                        width: isMobile ? 64.w : 72,
                        height: isMobile ? 64.w : 72,
                        decoration: BoxDecoration(
                          color: theme.colorScheme.primaryContainer,
                          borderRadius:
                              BorderRadius.circular(isMobile ? 18.r : 20),
                        ),
                        child: Icon(
                          Icons.warehouse_outlined,
                          size: isMobile ? 32.sp : 36,
                          color: theme.colorScheme.primary,
                        ),
                      ),
                      SizedBox(height: isMobile ? 20.h : 24),
                      Text(
                        loc.appName,
                        style: TextStyle(
                          fontSize: isMobile ? 22.sp : 24,
                          fontWeight: FontWeight.bold,
                          color: theme.colorScheme.onSurface,
                        ),
                      ),
                      SizedBox(height: isMobile ? 4.h : 6),
                      Text(
                        loc.loginForMore,
                        style: TextStyle(
                          fontSize: isMobile ? 13.sp : 14,
                          color: theme.colorScheme.onSurface.withAlpha(140),
                        ),
                      ),
                      SizedBox(height: isMobile ? 32.h : 40),

                      // 租户编号（可空）
                      TextFormField(
                        controller: _tenantController,
                        decoration: InputDecoration(
                          labelText: loc.tenantCode,
                          hintText: loc.tenantCodeHint,
                          prefixIcon:
                              const Icon(Icons.domain_outlined),
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(
                              isMobile ? 12.r : 14,
                            ),
                          ),
                        ),
                      ),
                      SizedBox(height: isMobile ? 16.h : 18),

                      // 用户名
                      TextFormField(
                        controller: _usernameController,
                        decoration: InputDecoration(
                          labelText: loc.username,
                          hintText: loc.usernameHint,
                          prefixIcon: const Icon(Icons.person_outline),
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(
                              isMobile ? 12.r : 14,
                            ),
                          ),
                        ),
                        validator: (value) {
                          if (value == null || value.trim().isEmpty) {
                            return loc.usernameHint;
                          }
                          return null;
                        },
                      ),
                      SizedBox(height: isMobile ? 16.h : 18),

                      // 密码
                      TextFormField(
                        controller: _passwordController,
                        obscureText: _obscurePassword,
                        decoration: InputDecoration(
                          labelText: loc.password,
                          hintText: loc.passwordHint,
                          prefixIcon: const Icon(Icons.lock_outline),
                          suffixIcon: IconButton(
                            icon: Icon(
                              _obscurePassword
                                  ? Icons.visibility_off_outlined
                                  : Icons.visibility_outlined,
                            ),
                            onPressed: () {
                              setState(() {
                                _obscurePassword = !_obscurePassword;
                              });
                            },
                          ),
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(
                              isMobile ? 12.r : 14,
                            ),
                          ),
                        ),
                        validator: (value) {
                          if (value == null || value.isEmpty) {
                            return loc.passwordHint;
                          }
                          return null;
                        },
                        onFieldSubmitted: (_) => _submit(),
                      ),
                      SizedBox(height: isMobile ? 28.h : 32),

                      // 提交按钮 + 状态提示（仅此部分随 Cubit 状态重建）
                      BlocBuilder<LoginCubit, LoginState>(
                        builder: (context, state) {
                          final loading = state is LoginLoading;
                          final failed = state is LoginFailure;
                          return Column(
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              if (failed)
                                Padding(
                                  padding: EdgeInsets.only(
                                    bottom: isMobile ? 12.h : 14,
                                  ),
                                  child: Text(
                                    loc.loginFailed,
                                    style: TextStyle(
                                      color: theme.colorScheme.error,
                                      fontSize: isMobile ? 13.sp : 13,
                                    ),
                                  ),
                                ),
                              SizedBox(
                                height: isMobile ? 48.h : 50,
                                child: FilledButton(
                                  onPressed: loading ? null : _submit,
                                  style: FilledButton.styleFrom(
                                    shape: RoundedRectangleBorder(
                                      borderRadius: BorderRadius.circular(
                                        isMobile ? 12.r : 14,
                                      ),
                                    ),
                                  ),
                                  child: loading
                                      ? SizedBox(
                                          width: isMobile ? 20.sp : 20,
                                          height: isMobile ? 20.sp : 20,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                            color:
                                                theme.colorScheme.onPrimary,
                                          ),
                                        )
                                      : Text(
                                          loc.loginButton,
                                          style: TextStyle(
                                            fontSize:
                                                isMobile ? 16.sp : 16,
                                            fontWeight: FontWeight.w600,
                                          ),
                                        ),
                                ),
                              ),
                            ],
                          );
                        },
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
