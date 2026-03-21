// lib/providers/settings_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/settings_state.dart';
import '../services/hive_service.dart';

final settingsProvider =
    StateNotifierProvider<SettingsNotifier, SettingsState>(
        (ref) => SettingsNotifier());

class SettingsNotifier extends StateNotifier<SettingsState> {
  SettingsNotifier() : super(const SettingsState()) {
    _load();
  }

  void _load() {
    try {
      final prefs = HiveService.preferences;
      final lang = prefs.get('language') as String? ?? 'en';
      final notif = prefs.get('notificationsEnabled') as bool? ?? true;
      state = SettingsState(language: lang, notificationsEnabled: notif);
    } catch (_) {
      // Hive not initialized (e.g., in tests) — keep defaults
    }
  }

  void setLanguage(String language) {
    state = state.copyWith(language: language);
    try {
      HiveService.preferences.put('language', language);
    } catch (_) {
      // Hive not initialized (e.g., in tests) — state update already applied
    }
  }

  void toggleNotifications() {
    final toggled = !state.notificationsEnabled;
    state = state.copyWith(notificationsEnabled: toggled);
    try {
      HiveService.preferences.put('notificationsEnabled', toggled);
    } catch (_) {
      // Hive not initialized (e.g., in tests) — state update already applied
    }
  }
}
