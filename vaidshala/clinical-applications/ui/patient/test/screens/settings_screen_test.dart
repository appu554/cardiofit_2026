// test/screens/settings_screen_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vaidshala_patient/screens/settings_screen.dart';
import 'package:vaidshala_patient/providers/settings_provider.dart';
import 'package:vaidshala_patient/models/settings_state.dart';

void main() {
  group('SettingsScreen', () {
    testWidgets('renders Account section with patient name', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            settingsProvider.overrideWith(
              (ref) => _FakeSettingsNotifier(),
            ),
          ],
          child: const MaterialApp(home: SettingsScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Settings'), findsOneWidget);
      expect(find.text('Rajesh Kumar'), findsOneWidget);
      expect(find.textContaining('+91'), findsOneWidget);
    });

    testWidgets('renders language dropdown', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            settingsProvider.overrideWith(
              (ref) => _FakeSettingsNotifier(),
            ),
          ],
          child: const MaterialApp(home: SettingsScreen()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Language'), findsOneWidget);
      expect(find.text('English'), findsOneWidget);
    });
  });
}

class _FakeSettingsNotifier extends StateNotifier<SettingsState>
    implements SettingsNotifier {
  _FakeSettingsNotifier() : super(const SettingsState());

  @override
  void setLanguage(String language) =>
      state = state.copyWith(language: language);

  @override
  void toggleNotifications() =>
      state = state.copyWith(notificationsEnabled: !state.notificationsEnabled);
}
