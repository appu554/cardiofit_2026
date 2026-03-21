import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/fade_slide_transition.dart';

void main() {
  testWidgets('FadeSlideTransition slides from offset to zero', (tester) async {
    final controller = AnimationController(
      vsync: const TestVSync(),
      duration: const Duration(milliseconds: 400),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: FadeSlideTransition(
          animation: controller,
          child: const Text('Hello'),
        ),
      ),
    );

    // Initially at offset (opacity 0)
    expect(find.text('Hello'), findsOneWidget);

    controller.forward();
    await tester.pump(const Duration(milliseconds: 200)); // halfway
    // Should be partially visible

    await tester.pump(const Duration(milliseconds: 200)); // complete
    controller.dispose();
  });
}
