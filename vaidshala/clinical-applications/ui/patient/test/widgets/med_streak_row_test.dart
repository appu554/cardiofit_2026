// test/widgets/med_streak_row_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/med_streak_row.dart';

void main() {
  group('MedStreakRow', () {
    testWidgets('renders medication name and streak days', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: MedStreakRow(
              name: 'Metformin',
              streakDays: 14,
            ),
          ),
        ),
      );

      expect(find.text('Metformin'), findsOneWidget);
      expect(find.text('14-day streak'), findsOneWidget);
    });

    testWidgets('shows check_circle icon', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: MedStreakRow(
              name: 'Lisinopril',
              streakDays: 7,
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.check_circle), findsOneWidget);
    });

    testWidgets('handles singular day streak correctly', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: MedStreakRow(
              name: 'Aspirin',
              streakDays: 1,
            ),
          ),
        ),
      );

      expect(find.text('1-day streak'), findsOneWidget);
    });
  });
}
