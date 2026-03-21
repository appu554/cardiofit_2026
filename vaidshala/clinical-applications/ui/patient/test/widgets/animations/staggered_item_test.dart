import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/staggered_item.dart';

void main() {
  testWidgets('StaggeredItem with index 0 starts immediately', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: StaggeredItem(
          index: 0,
          child: Text('Item 0'),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('Item 0'), findsOneWidget);
  });

  testWidgets('StaggeredItem beyond kMaxStaggerItems shows instantly', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: StaggeredItem(
          index: 100,
          child: Text('Item 100'),
        ),
      ),
    );
    // Should be visible immediately (no animation)
    expect(find.text('Item 100'), findsOneWidget);
  });
}
