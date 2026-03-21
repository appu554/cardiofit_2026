import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/count_up_text.dart';

void main() {
  testWidgets('CountUpText animates from 0 to value', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: CountUpText(value: 75, suffix: '%'),
      ),
    );

    // Initially at 0
    expect(find.text('0%'), findsOneWidget);

    // After animation completes
    await tester.pump(const Duration(milliseconds: 800));
    expect(find.text('75%'), findsOneWidget);
  });
}
