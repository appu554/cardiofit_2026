import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/providers/timeline_provider.dart';
import 'package:vaidshala_patient/providers/actions_provider.dart';
import 'package:vaidshala_patient/providers/auth_provider.dart';

void main() {
  group('MyDayProvider', () {
    late ProviderContainer container;

    setUp(() {
      container = ProviderContainer(
        overrides: [
          authStateProvider.overrideWith(() => _MockAuthNotifier()),
        ],
      );
    });

    tearDown(() => container.dispose());

    test('derives timeline entries from actions provider', () async {
      // Wait for actionsProvider to finish loading (API call fails → mock data)
      // Poll until isLoading is false, up to 2 seconds.
      for (var i = 0; i < 20; i++) {
        final actionsState = container.read(actionsProvider);
        if (!actionsState.isLoading) break;
        await Future.delayed(const Duration(milliseconds: 100));
      }

      final state = container.read(myDayProvider);
      expect(state.isLoading, false);
      expect(state.entries, isNotEmpty);

      // Entries should be sorted by time
      for (var i = 0; i < state.entries.length - 1; i++) {
        expect(
          state.entries[i].time.compareTo(state.entries[i + 1].time) <= 0,
          true,
        );
      }
    });
  });
}

class _MockAuthNotifier extends AuthNotifier {
  @override
  Future<AuthState> build() async =>
      const AuthState.authenticated(patientId: 'test-rajesh');
}
