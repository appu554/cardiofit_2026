import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/education_tip_card.dart';

void main() {
  testWidgets('EducationTipCard shows tip text', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: EducationTipCard(
            tip: 'Post-meal walks can reduce blood glucose by 15-20%',
          ),
        ),
      ),
    );

    expect(
        find.text('Post-meal walks can reduce blood glucose by 15-20%'),
        findsOneWidget);
  });
}
