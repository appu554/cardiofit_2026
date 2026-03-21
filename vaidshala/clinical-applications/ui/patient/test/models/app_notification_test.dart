import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/app_notification.dart';

void main() {
  group('AppNotification', () {
    test('creates with default read=false', () {
      final n = AppNotification(
        id: 'n1',
        type: NotificationType.coaching,
        title: 'Great progress!',
        body: 'Your FBG dropped 12 mg/dL this week',
        timestamp: DateTime(2026, 3, 21, 9, 0),
      );
      expect(n.read, false);
      expect(n.type, NotificationType.coaching);
    });

    test('serializes to/from JSON', () {
      final n = AppNotification(
        id: 'n2',
        type: NotificationType.alert,
        title: 'Test',
        body: 'Body',
        deepLink: '/home/progress',
        timestamp: DateTime(2026, 3, 21),
        read: true,
      );
      final json = n.toJson();
      final restored = AppNotification.fromJson(json);
      expect(restored, n);
    });
  });
}
