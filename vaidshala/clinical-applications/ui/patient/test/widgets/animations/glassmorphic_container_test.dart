import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/glassmorphic_container.dart';

void main() {
  testWidgets('GlassmorphicContainer renders child', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: GlassmorphicContainer(
          child: Text('Sheet content'),
        ),
      ),
    );
    expect(find.text('Sheet content'), findsOneWidget);
  });
}
