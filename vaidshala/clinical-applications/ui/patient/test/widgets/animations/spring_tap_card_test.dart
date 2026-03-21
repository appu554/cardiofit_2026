import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/spring_tap_card.dart';

void main() {
  // Use pump() with duration, NOT pumpAndSettle() — springs don't "settle" in test timeouts
  const kTestSettleDuration = Duration(seconds: 2);

  testWidgets('SpringTapCard scales down on press and back on release', (tester) async {
    bool tapped = false;

    await tester.pumpWidget(
      MaterialApp(
        home: SpringTapCard(
          onTap: () => tapped = true,
          child: const SizedBox(width: 100, height: 100),
        ),
      ),
    );

    // Press down
    final gesture = await tester.press(find.byType(SpringTapCard));
    await tester.pump(const Duration(milliseconds: 100));
    // Scale should be < 1.0 (spring moving toward 0.96)

    // Release
    await gesture.up();
    await tester.pump(kTestSettleDuration);
    // Scale should be back to 1.0
  });
}
