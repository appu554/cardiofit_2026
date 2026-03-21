// test/providers/notifications_provider_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/models/app_notification.dart';
import 'package:vaidshala_patient/providers/notifications_provider.dart';

void main() {
  group('notificationsProvider', () {
    test('mock seed data returns 5 notifications', () async {
      final container = ProviderContainer(
        overrides: [
          notificationsProvider.overrideWith(() => _MockNotificationsNotifier()),
        ],
      );
      addTearDown(container.dispose);

      // Wait for async build
      await container.read(notificationsProvider.future);
      final state = container.read(notificationsProvider).valueOrNull ?? [];
      expect(state.length, 5);
    });

    test('unread count derived from notifications', () async {
      final mockNotifications = [
        AppNotification(
          id: 'n1',
          type: NotificationType.coaching,
          title: 'Test',
          body: 'Body',
          timestamp: DateTime.now(),
          read: true,
        ),
        AppNotification(
          id: 'n2',
          type: NotificationType.alert,
          title: 'Test2',
          body: 'Body2',
          timestamp: DateTime.now(),
          read: false,
        ),
      ];

      final container = ProviderContainer(
        overrides: [
          notificationsProvider.overrideWith(
            () => _FixedNotificationsNotifier(mockNotifications),
          ),
        ],
      );
      addTearDown(container.dispose);

      await container.read(notificationsProvider.future);
      final unread = container.read(unreadCountProvider);
      expect(unread, 1);
    });
  });
}

class _MockNotificationsNotifier extends AsyncNotifier<List<AppNotification>>
    implements NotificationsNotifier {
  @override
  Future<List<AppNotification>> build() async {
    return _mockSeedData();
  }

  @override
  Future<void> markRead(String id) async {
    final current = state.valueOrNull ?? [];
    state = AsyncData(
      current.map((n) => n.id == id ? n.copyWith(read: true) : n).toList(),
    );
  }

  @override
  Future<void> markAllRead() async {
    final current = state.valueOrNull ?? [];
    state = AsyncData(current.map((n) => n.copyWith(read: true)).toList());
  }

  @override
  Future<void> dismiss(String id) async {
    final current = state.valueOrNull ?? [];
    state = AsyncData(current.where((n) => n.id != id).toList());
  }

  @override
  Future<void> refresh() async {
    state = AsyncData(await build());
  }
}

class _FixedNotificationsNotifier extends AsyncNotifier<List<AppNotification>>
    implements NotificationsNotifier {
  final List<AppNotification> _data;
  _FixedNotificationsNotifier(this._data);

  @override
  Future<List<AppNotification>> build() async => _data;

  @override
  Future<void> markRead(String id) async {}
  @override
  Future<void> markAllRead() async {}
  @override
  Future<void> dismiss(String id) async {}
  @override
  Future<void> refresh() async {}
}

List<AppNotification> _mockSeedData() {
  final now = DateTime.now();
  return [
    AppNotification(
      id: 'n1', type: NotificationType.coaching,
      title: 'Great progress!', body: 'Your FBG dropped 12 mg/dL this week',
      deepLink: '/home/progress', timestamp: now.copyWith(hour: 9), read: true,
    ),
    AppNotification(
      id: 'n2', type: NotificationType.alert,
      title: 'FBG trending down', body: 'Your fasting glucose is moving toward target',
      deepLink: '/home/progress', timestamp: now.copyWith(hour: 8), read: false,
    ),
    AppNotification(
      id: 'n3', type: NotificationType.reminder,
      title: 'Time for evening walk', body: 'A 15-min post-dinner walk can lower glucose by 15-20%',
      deepLink: '/home/my-day', timestamp: now.copyWith(hour: 19), read: false,
    ),
    AppNotification(
      id: 'n4', type: NotificationType.coaching,
      title: 'Weekly progress summary', body: 'You completed 85% of actions this week',
      deepLink: '/home/progress', timestamp: now.subtract(const Duration(days: 1)), read: true,
    ),
    AppNotification(
      id: 'n5', type: NotificationType.milestone,
      title: 'New health tip available', body: "Learn about protein's role in metabolic health",
      deepLink: '/home/learn', timestamp: now.subtract(const Duration(days: 3)), read: true,
    ),
  ];
}
