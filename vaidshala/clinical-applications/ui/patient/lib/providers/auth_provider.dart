import 'package:flutter_riverpod/flutter_riverpod.dart';

class AuthState {
  final bool isAuthenticated;
  final String? patientId;
  final String? phone;

  const AuthState({
    this.isAuthenticated = false,
    this.patientId,
    this.phone,
  });

  AuthState copyWith({
    bool? isAuthenticated,
    String? patientId,
    String? phone,
  }) =>
      AuthState(
        isAuthenticated: isAuthenticated ?? this.isAuthenticated,
        patientId: patientId ?? this.patientId,
        phone: phone ?? this.phone,
      );
}

final authStateProvider =
    AsyncNotifierProvider<AuthNotifier, AuthState>(AuthNotifier.new);

class AuthNotifier extends AsyncNotifier<AuthState> {
  @override
  Future<AuthState> build() async {
    // Dev mode: auto-authenticate with mock patient
    return const AuthState(
      isAuthenticated: true,
      patientId: 'rajesh-kumar-001',
    );
  }

  Future<void> login(String phone) async {
    state = AsyncData(AuthState(isAuthenticated: true, phone: phone, patientId: 'rajesh-kumar-001'));
  }

  Future<void> verifyOtp(String otp) async {
    final current = state.valueOrNull ?? const AuthState();
    state = AsyncData(current.copyWith(isAuthenticated: true, patientId: 'rajesh-kumar-001'));
  }

  void logout() {
    state = const AsyncData(AuthState());
  }
}
