// @TestOn('browser') — Drift WASM requires dart:js_interop (browser-only).
// This test must be run with `flutter test --platform chrome` or
// `flutter drive` on a web target. It is excluded from `flutter test` on
// the native VM runner intentionally.
@TestOn('browser')
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/timeline_entry.dart';
import 'package:vaidshala_patient/providers/timeline_provider.dart';
import 'package:vaidshala_patient/screens/my_day_tab.dart';

void main() {
  testWidgets('MyDayTab renders header', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          myDayProvider.overrideWithValue(
            const MyDayState(
              entries: [
                TimelineEntry(
                  id: 'act-001',
                  time: '06:30',
                  text: 'Measure fasting blood glucose',
                  icon: 'bloodtype',
                  done: true,
                ),
              ],
              tipOfTheDay: 'Walking after meals helps lower blood sugar',
            ),
          ),
        ],
        child: const MaterialApp(home: Scaffold(body: MyDayTab())),
      ),
    );

    await tester.pump();

    expect(find.text('My Day'), findsOneWidget);
    expect(find.text('Your daily health routine'), findsOneWidget);
  });

  testWidgets('MyDayTab has SpeedDial FAB', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          myDayProvider.overrideWithValue(
            const MyDayState(entries: [], tipOfTheDay: null),
          ),
        ],
        child: const MaterialApp(home: Scaffold(body: MyDayTab())),
      ),
    );

    await tester.pump();

    // Main FAB is present
    expect(find.byIcon(Icons.add), findsOneWidget);
  });
}
