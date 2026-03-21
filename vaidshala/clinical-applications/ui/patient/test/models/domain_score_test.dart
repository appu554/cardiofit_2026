import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/domain_score.dart';

void main() {
  group('DomainScore', () {
    test('creates with required fields', () {
      const ds = DomainScore(
        name: 'Blood Sugar',
        score: 35,
        target: 60,
        icon: 'bloodtype',
      );
      expect(ds.name, 'Blood Sugar');
      expect(ds.score, 35);
      expect(ds.target, 60);
    });

    test('serializes to/from JSON', () {
      const ds = DomainScore(
        name: 'Activity',
        score: 22,
        target: 50,
        icon: 'directions_walk',
      );
      final json = ds.toJson();
      final restored = DomainScore.fromJson(json);
      expect(restored, ds);
    });
  });
}
