import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/cause_effect.dart';

void main() {
  group('CauseEffect', () {
    test('creates with required fields', () {
      const ce = CauseEffect(
        id: 'ce-1',
        cause: 'Walking 4000 steps daily',
        effect: 'FBG dropped 12 mg/dL',
        causeIcon: 'directions_walk',
        effectIcon: 'bloodtype',
      );
      expect(ce.verified, false);
    });

    test('serializes to/from JSON', () {
      const ce = CauseEffect(
        id: 'ce-1',
        cause: 'Walking',
        effect: 'FBG dropped',
        causeIcon: 'directions_walk',
        effectIcon: 'bloodtype',
        verified: true,
      );
      expect(CauseEffect.fromJson(ce.toJson()), ce);
    });
  });
}
