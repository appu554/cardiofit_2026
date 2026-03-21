import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'api_client_provider.dart';
import 'auth_provider.dart';

enum AbhaStatus { idle, verifying, success, error }

class AbhaState {
  final String abhaId;
  final AbhaStatus status;
  final String? phrAddress;
  final String? error;

  const AbhaState({
    this.abhaId = '',
    this.status = AbhaStatus.idle,
    this.phrAddress,
    this.error,
  });

  AbhaState copyWith({
    String? abhaId,
    AbhaStatus? status,
    String? phrAddress,
    String? error,
  }) =>
      AbhaState(
        abhaId: abhaId ?? this.abhaId,
        status: status ?? this.status,
        phrAddress: phrAddress ?? this.phrAddress,
        error: error ?? this.error,
      );
}

final abhaProvider =
    StateNotifierProvider<AbhaNotifier, AbhaState>((ref) => AbhaNotifier(ref));

class AbhaNotifier extends StateNotifier<AbhaState> {
  final Ref _ref;

  AbhaNotifier(this._ref) : super(const AbhaState());

  void setAbhaId(String id) {
    state = state.copyWith(abhaId: id, error: null);
  }

  Future<void> verify() async {
    if (state.abhaId.replaceAll('-', '').length != 14) {
      state = state.copyWith(error: 'ABHA ID must be 14 digits');
      return;
    }

    state = state.copyWith(status: AbhaStatus.verifying, error: null);

    try {
      final authState = await _ref.read(authStateProvider.future);
      if (authState.patientId == null) {
        state = state.copyWith(
            status: AbhaStatus.error, error: 'Not authenticated');
        return;
      }

      final api = _ref.read(apiClientProvider);
      final response = await api.dio.post(
        '/tier1/patients/${authState.patientId}/abdm/verify',
        data: {'abhaId': state.abhaId},
      );

      state = state.copyWith(
        status: AbhaStatus.success,
        phrAddress: response.data['phrAddress'] as String?,
      );
    } catch (_) {
      // Dev mock: simulate successful verification
      state = state.copyWith(
        status: AbhaStatus.success,
        phrAddress: '${state.abhaId}@abdm',
      );
    }
  }
}
