import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/cause_effect.dart';
import 'package:vaidshala_patient/widgets/cause_effect_card.dart';

void main() {
  testWidgets('CauseEffectCard shows cause and effect text', (tester) async {
    const ce = CauseEffect(
      id: 'ce-1',
      cause: 'Taking Metformin consistently',
      effect: 'FBG dropped from 192 to 178 mg/dL',
      causeIcon: 'medication',
      effectIcon: 'bloodtype',
      verified: true,
    );

    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: CauseEffectCard(causeEffect: ce))),
    );

    expect(find.text('Taking Metformin consistently'), findsOneWidget);
    expect(find.text('FBG dropped from 192 to 178 mg/dL'), findsOneWidget);
    expect(find.byIcon(Icons.verified), findsOneWidget);
  });

  testWidgets('hides verified icon when not verified', (tester) async {
    const ce = CauseEffect(
      id: 'ce-2',
      cause: 'Walking',
      effect: 'BP rising',
      causeIcon: 'directions_walk',
      effectIcon: 'favorite',
      verified: false,
    );

    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: CauseEffectCard(causeEffect: ce))),
    );

    expect(find.byIcon(Icons.verified), findsNothing);
  });
}
