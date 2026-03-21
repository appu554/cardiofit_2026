// test/widgets/symptom_chip_grid_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/symptom_chip_grid.dart';

void main() {
  group('SymptomChip', () {
    testWidgets('renders label text', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SymptomChip(
              label: 'Dizziness',
              selected: false,
              onTap: () {},
            ),
          ),
        ),
      );

      expect(find.text('Dizziness'), findsOneWidget);
    });

    testWidgets('shows check icon when selected', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SymptomChip(
              label: 'Fatigue',
              selected: true,
              onTap: () {},
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.check), findsOneWidget);
    });

    testWidgets('calls onTap when tapped', (tester) async {
      var tapped = false;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SymptomChip(
              label: 'Nausea',
              selected: false,
              onTap: () => tapped = true,
            ),
          ),
        ),
      );

      await tester.tap(find.text('Nausea'));
      expect(tapped, true);
    });
  });

  group('SymptomChipGrid', () {
    testWidgets('renders all symptom chips', (tester) async {
      final symptoms = ['Dizziness', 'Nausea', 'Fatigue'];
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SymptomChipGrid(
              symptoms: symptoms,
              selected: const {},
              onToggle: (_) {},
            ),
          ),
        ),
      );

      expect(find.text('Dizziness'), findsOneWidget);
      expect(find.text('Nausea'), findsOneWidget);
      expect(find.text('Fatigue'), findsOneWidget);
    });

    testWidgets('shows selected state for selected symptoms', (tester) async {
      final symptoms = ['Dizziness', 'Nausea'];
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SymptomChipGrid(
              symptoms: symptoms,
              selected: const {'Dizziness'},
              onToggle: (_) {},
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.check), findsOneWidget);
    });
  });
}
