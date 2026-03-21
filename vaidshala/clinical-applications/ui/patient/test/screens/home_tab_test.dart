// test/screens/home_tab_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/driver.dart';
import 'package:vaidshala_patient/models/health_score.dart';
import 'package:vaidshala_patient/models/insight.dart';
import 'package:vaidshala_patient/providers/actions_provider.dart';
import 'package:vaidshala_patient/providers/drivers_provider.dart';
import 'package:vaidshala_patient/providers/health_score_provider.dart';
import 'package:vaidshala_patient/providers/insights_provider.dart';
import 'package:vaidshala_patient/screens/home_tab.dart';

void main() {
  group('HomeTab', () {
    testWidgets('renders greeting text', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            healthScoreProvider.overrideWith(
              () => _FakeHealthScoreNotifier(),
            ),
            actionsProvider.overrideWith(
              (ref) => _FakeActionsNotifier(ref),
            ),
            healthDriversProvider.overrideWith(
              (ref) async => <HealthDriver>[],
            ),
            insightsProvider.overrideWith(
              (ref) async => const Insight(),
            ),
          ],
          child: const MaterialApp(home: HomeTab()),
        ),
      );

      // Let async providers resolve
      await tester.pump();

      expect(find.textContaining('Namaste'), findsOneWidget);
    });
  });
}

class _FakeHealthScoreNotifier extends HealthScoreNotifier {
  @override
  Future<HealthScore?> build() async {
    return const HealthScore(score: 72, label: 'Good', delta: 3);
  }
}

class _FakeActionsNotifier extends ActionsNotifier {
  _FakeActionsNotifier(super.ref);

  @override
  ActionsState get state => const ActionsState();
}
