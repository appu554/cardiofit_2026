// test/screens/notifications_screen_test.dart
//
// NOTE: These tests require the chrome platform due to drift/wasm transitively
// importing dart:js_interop (unavailable on native). Run with:
//   flutter test test/screens/notifications_screen_test.dart --platform chrome
//
// ignore_for_file: avoid_implementing_value_types
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/models/app_notification.dart';
import 'package:vaidshala_patient/providers/notifications_provider.dart';
import 'package:vaidshala_patient/screens/notifications_screen.dart';

void main() {
  group('NotificationsScreen', () {
    testWidgets('renders empty state when no notifications', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            notificationsProvider.overrideWith(
              () => _EmptyNotificationsNotifier(),
            ),
          ],
          child: const MaterialApp(home: NotificationsScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('No notifications yet'), findsOneWidget);
    });

    testWidgets('renders notifications grouped by date', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            notificationsProvider.overrideWith(
              () => _MockNotificationsNotifier(),
            ),
          ],
          child: const MaterialApp(home: NotificationsScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Great progress!'), findsOneWidget);
      expect(find.text('TODAY'), findsOneWidget);
    });
  });
}

class _EmptyNotificationsNotifier extends AsyncNotifier<List<AppNotification>>
    implements NotificationsNotifier {
  @override
  Future<List<AppNotification>> build() async => [];
  @override
  Future<void> markRead(String id) async {}
  @override
  Future<void> markAllRead() async {}
  @override
  Future<void> dismiss(String id) async {}
  @override
  Future<void> refresh() async {}
}

class _MockNotificationsNotifier extends AsyncNotifier<List<AppNotification>>
    implements NotificationsNotifier {
  @override
  Future<List<AppNotification>> build() async => [
        AppNotification(
          id: 'n1',
          type: NotificationType.coaching,
          title: 'Great progress!',
          body: 'Your FBG dropped',
          timestamp: DateTime.now(),
          read: true,
        ),
      ];
  @override
  Future<void> markRead(String id) async {}
  @override
  Future<void> markAllRead() async {}
  @override
  Future<void> dismiss(String id) async {}
  @override
  Future<void> refresh() async {}
}
