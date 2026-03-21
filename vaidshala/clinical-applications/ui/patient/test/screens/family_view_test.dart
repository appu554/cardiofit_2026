import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/family_view_data.dart';
import 'package:vaidshala_patient/providers/family_view_provider.dart';
import 'package:vaidshala_patient/screens/family_view_screen.dart';

void main() {
  testWidgets('FamilyViewScreen renders with mock data', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          familyViewProvider('test-token').overrideWith(
            (ref) async => const FamilyViewData(
              patientName: 'Rajesh',
              mealActions: [
                FamilyAction(
                  text: 'Prepare low-GI breakfast',
                  icon: 'restaurant',
                  time: '07:30',
                ),
              ],
              activityActions: [
                FamilyAction(
                  text: 'Encourage a walk after dinner',
                  icon: 'directions_walk',
                  time: '20:00',
                ),
              ],
              supportMessage: 'Your support makes a difference!',
            ),
          ),
        ],
        child: const MaterialApp(
            home: FamilyViewScreen(token: 'test-token')),
      ),
    );

    // Wait for FutureProvider to resolve
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump();

    expect(find.textContaining('Health Plan Today'), findsOneWidget);
    expect(find.text('Powered by Vaidshala'), findsOneWidget);
  });
}
