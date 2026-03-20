import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/milestone.dart';

void main() {
  group('Milestone', () {
    test('achieved milestone has date', () {
      const m = Milestone(
        id: 'm-1',
        title: 'First Week Complete',
        description: 'Completed 7 days of tracking',
        status: MilestoneStatus.achieved,
        achievedDate: '2026-03-15',
      );
      expect(m.status, MilestoneStatus.achieved);
      expect(m.achievedDate, isNotNull);
    });

    test('locked milestone has no date', () {
      const m = Milestone(
        id: 'm-2',
        title: 'FBG Below 140',
        description: 'Reach fasting glucose target',
        status: MilestoneStatus.locked,
      );
      expect(m.achievedDate, isNull);
    });
  });
}
