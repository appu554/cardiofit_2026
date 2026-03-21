import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/alert_card.dart';

void main() {
  testWidgets('AlertCard shows message and gentle reminder label',
      (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: AlertCard(
              message: 'FBG trending high at 178 mg/dL'),
        ),
      ),
    );

    expect(find.text('Gentle Reminder'), findsOneWidget);
    expect(find.text('FBG trending high at 178 mg/dL'), findsOneWidget);
    expect(find.byIcon(Icons.info_outline), findsOneWidget);
  });

  testWidgets('AlertCard has accessibility label', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: AlertCard(message: 'Test alert'),
        ),
      ),
    );

    final semantics = tester.getSemantics(find.byType(AlertCard));
    expect(semantics.label, contains('Important health reminder'));
  });
}
