// test/widgets/notification_item_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/app_notification.dart';
import 'package:vaidshala_patient/widgets/notification_item.dart';

void main() {
  group('NotificationItem', () {
    testWidgets('renders title and body', (tester) async {
      final notification = AppNotification(
        id: 'n1',
        type: NotificationType.coaching,
        title: 'Great progress!',
        body: 'Your FBG dropped 12 mg/dL',
        timestamp: DateTime.now(),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: NotificationItem(
              notification: notification,
              onTap: () {},
              onDismiss: () {},
            ),
          ),
        ),
      );

      expect(find.text('Great progress!'), findsOneWidget);
      expect(find.text('Your FBG dropped 12 mg/dL'), findsOneWidget);
    });

    testWidgets('shows unread dot when not read', (tester) async {
      final notification = AppNotification(
        id: 'n2',
        type: NotificationType.alert,
        title: 'Alert',
        body: 'Body',
        timestamp: DateTime.now(),
        read: false,
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: NotificationItem(
              notification: notification,
              onTap: () {},
              onDismiss: () {},
            ),
          ),
        ),
      );

      // Unread dot is a small blue Container
      final dot = find.byWidgetPredicate(
        (w) => w is Container && w.decoration is BoxDecoration &&
               (w.decoration as BoxDecoration).color == Colors.blue,
      );
      expect(dot, findsOneWidget);
    });
  });
}
