import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/providers/progress_provider.dart';
import 'package:vaidshala_patient/providers/auth_provider.dart';
import 'package:vaidshala_patient/models/milestone.dart';

void main() {
  group('ProgressProvider', () {
    late ProviderContainer container;

    setUp(() {
      // Override auth to avoid SecureStorageService hitting IndexedDB in tests
      container = ProviderContainer(
        overrides: [
          authStateProvider.overrideWith(() => _MockAuthNotifier()),
        ],
      );
    });

    tearDown(() => container.dispose());

    test('returns mock data with 6 metrics when API unavailable', () async {
      final state = await container.read(progressProvider.future);
      expect(state.metrics.length, 6);
      expect(state.metrics.first.id, 'fbg');
      expect(state.metrics.first.current, 178);
    });

    test('returns 3 cause-effect items', () async {
      final state = await container.read(progressProvider.future);
      expect(state.causeEffects.length, 3);
      expect(state.causeEffects.first.verified, true);
    });

    test('returns 5 milestones (2 achieved, 3 locked)', () async {
      final state = await container.read(progressProvider.future);
      expect(state.milestones.length, 5);
      final achieved =
          state.milestones.where((m) => m.status == MilestoneStatus.achieved);
      expect(achieved.length, 2);
    });
  });
}

// Mock that returns authenticated state with a fake patientId — forces API call
// which fails (no server running in test) → catch block returns mock data
class _MockAuthNotifier extends AuthNotifier {
  @override
  Future<AuthState> build() async => const AuthState.authenticated(
        patientId: 'test-patient-rajesh',
      );
}
