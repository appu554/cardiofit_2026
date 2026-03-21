// test/screens/progress_tab_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/cause_effect.dart';
import 'package:vaidshala_patient/models/milestone.dart';
import 'package:vaidshala_patient/models/progress_metric.dart';
import 'package:vaidshala_patient/providers/progress_provider.dart';
import 'package:vaidshala_patient/screens/progress_tab.dart';

class _FakeProgressNotifier extends ProgressNotifier {
  @override
  Future<ProgressState> build() async => const ProgressState(
        metrics: [
          ProgressMetric(
            id: 'fbg',
            name: 'Fasting Blood Glucose',
            icon: 'bloodtype',
            current: 178,
            previous: 185,
            target: 126,
            unit: 'mg/dL',
            improving: true,
            sparkline: [192, 188, 185, 182, 180, 178],
          ),
        ],
        causeEffects: [
          CauseEffect(
            id: 'ce-1',
            cause: 'Taking Metformin consistently',
            effect: 'FBG dropped from 192 to 178 mg/dL',
            causeIcon: 'medication',
            effectIcon: 'bloodtype',
            verified: true,
          ),
        ],
        milestones: [
          Milestone(
            id: 'm-1',
            title: 'First Week Complete',
            description: 'Completed 7 days of health tracking',
            status: MilestoneStatus.achieved,
            achievedDate: '2026-01-14',
          ),
        ],
      );
}

void main() {
  testWidgets('ProgressTab renders header text', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          progressProvider.overrideWith(() => _FakeProgressNotifier()),
        ],
        child: const MaterialApp(home: Scaffold(body: ProgressTab())),
      ),
    );

    // Let async provider resolve
    await tester.pump();

    expect(find.text('Your Progress'), findsOneWidget);
    expect(find.text('Key Metrics'), findsOneWidget);
  });

  testWidgets('ProgressTab shows Fasting Blood Glucose metric',
      (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          progressProvider.overrideWith(() => _FakeProgressNotifier()),
        ],
        child: const MaterialApp(home: Scaffold(body: ProgressTab())),
      ),
    );

    await tester.pump();

    expect(find.text('Fasting Blood Glucose'), findsOneWidget);
  });
}
