// test/widgets/bp_entry_card_test.dart
//
// NOTE: These tests require the chrome platform due to drift/wasm transitively
// importing dart:js_interop (unavailable on native). Run with:
//   flutter test test/widgets/bp_entry_card_test.dart --platform chrome
//
// ignore_for_file: avoid_implementing_value_types
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/widgets/bp_entry_card.dart';
import 'package:vaidshala_patient/providers/vitals_entry_provider.dart';

/// In-memory notifier that skips database access — safe for widget tests.
class _FakeVitalsNotifier extends VitalsEntryNotifier {
  _FakeVitalsNotifier(super.ref);

  @override
  Future<bool> saveBp() async {
    if (!validateBp()) return false;
    return true;
  }
}

void main() {
  Widget buildWidget(Widget child) => ProviderScope(
        overrides: [
          vitalsEntryProvider.overrideWith((ref) => _FakeVitalsNotifier(ref)),
        ],
        child: MaterialApp(home: Scaffold(body: child)),
      );

  group('BpEntryCard', () {
    testWidgets('renders systolic and diastolic fields', (tester) async {
      await tester.pumpWidget(buildWidget(BpEntryCard(onSaved: () {})));

      expect(find.text('Blood Pressure'), findsOneWidget);
      expect(find.byType(TextFormField), findsNWidgets(2));
      expect(find.text('Save'), findsOneWidget);
    });

    testWidgets('shows validation error for out-of-range systolic', (tester) async {
      await tester.pumpWidget(buildWidget(BpEntryCard(onSaved: () {})));

      await tester.enterText(find.byType(TextFormField).first, '300');
      await tester.enterText(find.byType(TextFormField).last, '80');
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();

      expect(find.textContaining('between 60 and 250'), findsOneWidget);
    });
  });
}
