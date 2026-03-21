// @TestOn('browser') — MainShell now imports notifications_provider which
// transitively imports Drift WASM (dart:js_interop, browser-only).
// Run with `flutter test --platform chrome` or on a web target.
@TestOn('browser')
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:vaidshala_patient/screens/main_shell.dart';

void main() {
  group('MainShell', () {
    // MainShell uses GoRouterState.of(context), so we must provide a
    // GoRouter ancestor in the widget tree.
    Widget buildWithRouter({String location = '/home/dashboard'}) {
      final router = GoRouter(
        initialLocation: location,
        routes: [
          ShellRoute(
            builder: (context, state, child) => MainShell(child: child),
            routes: [
              GoRoute(
                path: '/home/dashboard',
                pageBuilder: (context, state) => const NoTransitionPage(
                  child: Center(child: Text('Dashboard Content')),
                ),
              ),
              GoRoute(
                path: '/home/progress',
                pageBuilder: (context, state) => const NoTransitionPage(
                  child: Center(child: Text('Progress Content')),
                ),
              ),
              GoRoute(
                path: '/home/my-day',
                pageBuilder: (context, state) => const NoTransitionPage(
                  child: Center(child: Text('My Day Content')),
                ),
              ),
              GoRoute(
                path: '/home/learn',
                pageBuilder: (context, state) => const NoTransitionPage(
                  child: Center(child: Text('Learn Content')),
                ),
              ),
            ],
          ),
        ],
      );

      return ProviderScope(
        child: MaterialApp.router(routerConfig: router),
      );
    }

    testWidgets('renders 4 navigation tabs', (tester) async {
      await tester.pumpWidget(buildWithRouter());
      await tester.pumpAndSettle();

      expect(find.text('Home'), findsOneWidget);
      expect(find.text('Progress'), findsOneWidget);
      expect(find.text('My Day'), findsOneWidget);
      expect(find.text('Learn'), findsOneWidget);
    });

    testWidgets('renders child content for dashboard route', (tester) async {
      await tester.pumpWidget(buildWithRouter());
      await tester.pumpAndSettle();

      expect(find.text('Dashboard Content'), findsOneWidget);
    });

    testWidgets('selects correct tab for progress route', (tester) async {
      await tester.pumpWidget(buildWithRouter(location: '/home/progress'));
      await tester.pumpAndSettle();

      expect(find.text('Progress Content'), findsOneWidget);
    });

    testWidgets('renders AppBar with Vaidshala title', (tester) async {
      await tester.pumpWidget(buildWithRouter());
      await tester.pumpAndSettle();

      expect(find.text('Vaidshala'), findsOneWidget);
    });
  });
}
