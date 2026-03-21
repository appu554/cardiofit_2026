// test/widgets/settings_tile_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/settings_tile.dart';

void main() {
  group('SettingsTile', () {
    testWidgets('renders icon, title, and trailing widget', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SettingsTile(
              icon: Icons.person,
              title: 'Account Name',
              trailing: Text('Rajesh Kumar'),
            ),
          ),
        ),
      );

      expect(find.byIcon(Icons.person), findsOneWidget);
      expect(find.text('Account Name'), findsOneWidget);
      expect(find.text('Rajesh Kumar'), findsOneWidget);
    });

    testWidgets('calls onTap when tapped', (tester) async {
      var tapped = false;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SettingsTile(
              icon: Icons.settings,
              title: 'Tap Me',
              onTap: () => tapped = true,
            ),
          ),
        ),
      );

      await tester.tap(find.text('Tap Me'));
      expect(tapped, true);
    });
  });
}
