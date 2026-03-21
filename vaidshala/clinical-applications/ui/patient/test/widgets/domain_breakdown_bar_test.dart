// test/widgets/domain_breakdown_bar_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/domain_breakdown_bar.dart';

void main() {
  group('DomainBreakdownBar', () {
    testWidgets('renders label, score, and target', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: DomainBreakdownBar(
              label: 'Blood Sugar',
              score: 35,
              target: 60,
              icon: Icons.bloodtype,
              color: Color(0xFFC62828),
            ),
          ),
        ),
      );

      expect(find.text('Blood Sugar'), findsOneWidget);
      expect(find.text('35'), findsOneWidget);
      expect(find.textContaining('60'), findsOneWidget);
    });
  });
}
