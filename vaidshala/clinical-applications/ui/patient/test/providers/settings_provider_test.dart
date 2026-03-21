// test/providers/settings_provider_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/providers/settings_provider.dart';
import 'package:vaidshala_patient/models/settings_state.dart';

void main() {
  group('SettingsNotifier', () {
    test('initial state has English and notifications enabled', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final state = container.read(settingsProvider);
      expect(state.language, 'en');
      expect(state.notificationsEnabled, true);
    });

    test('setLanguage updates language', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(settingsProvider.notifier).setLanguage('hi');
      expect(container.read(settingsProvider).language, 'hi');
    });

    test('toggleNotifications flips the flag', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(settingsProvider.notifier).toggleNotifications();
      expect(container.read(settingsProvider).notificationsEnabled, false);
    });
  });
}
