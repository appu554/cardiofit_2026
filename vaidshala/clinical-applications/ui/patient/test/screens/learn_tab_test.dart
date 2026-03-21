// test/screens/learn_tab_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/clinical_translation.dart';
import 'package:vaidshala_patient/providers/learn_provider.dart';
import 'package:vaidshala_patient/screens/learn_tab.dart';

void main() {
  testWidgets('LearnTab renders header and sections', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          learnProvider.overrideWithValue(
            const LearnState(
              alerts: ['FBG trending high'],
              tips: ['Walk after meals'],
              translations: [
                ClinicalTranslation(
                  clinicalTerm: 'HbA1c',
                  patientTerm: 'Average blood sugar over 3 months',
                  explanation: 'A blood test showing average sugar level.',
                ),
              ],
            ),
          ),
        ],
        child: const MaterialApp(home: Scaffold(body: LearnTab())),
      ),
    );

    await tester.pump();

    expect(find.text('Learn'), findsOneWidget);
    expect(find.text('Understanding Your Reports'), findsOneWidget);
  });

  testWidgets('LearnTab shows clinical translations', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          learnProvider.overrideWithValue(
            const LearnState(
              translations: [
                ClinicalTranslation(
                  clinicalTerm: 'HbA1c',
                  patientTerm: 'Average blood sugar over 3 months',
                  explanation: 'A blood test showing average sugar level.',
                ),
              ],
            ),
          ),
        ],
        child: const MaterialApp(home: Scaffold(body: LearnTab())),
      ),
    );

    await tester.pump();

    // ClinicalTranslationRow uses RichText for clinicalTerm; find via widget predicate
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is RichText &&
            widget.text.toPlainText().contains('HbA1c'),
      ),
      findsWidgets,
    );
  });
}
