// test/screens/score_detail_screen_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/screens/score_detail_screen.dart';
import 'package:vaidshala_patient/providers/score_detail_provider.dart';
import 'package:vaidshala_patient/models/domain_score.dart';

void main() {
  group('ScoreDetailScreen', () {
    testWidgets('renders domain breakdown bars', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            scoreDetailProvider.overrideWithValue(
              const ScoreDetailState(
                score: 18,
                label: 'Needs Attention',
                scoreHistory: [25.0, 28.0, 24.0, 22.0, 20.0, 19.0, 18.0, 18.0, 19.0, 17.0, 18.0, 18.0],
                domains: [
                  DomainScore(name: 'Blood Sugar', score: 35, target: 60, icon: 'bloodtype'),
                  DomainScore(name: 'Activity', score: 22, target: 50, icon: 'directions_walk'),
                ],
                explanation: 'Test explanation text',
              ),
            ),
          ],
          child: const MaterialApp(home: ScoreDetailScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Your Health Score'), findsOneWidget);
      expect(find.text('Blood Sugar'), findsOneWidget);
      expect(find.text('Activity'), findsOneWidget);
      expect(find.text('Test explanation text'), findsOneWidget);
    });
  });
}
