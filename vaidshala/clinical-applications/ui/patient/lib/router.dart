// lib/router.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'providers/auth_provider.dart';
import 'screens/splash_screen.dart';
import 'screens/onboarding_screen.dart';
import 'screens/login_screen.dart';
import 'screens/otp_screen.dart';
import 'screens/main_shell.dart';
import 'screens/home_tab.dart';
import 'screens/progress_tab.dart';
import 'screens/my_day_tab.dart';
import 'screens/learn_tab.dart';
import 'screens/abha_verification_screen.dart';
import 'screens/family_view_screen.dart';

final routerProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authStateProvider);

  return GoRouter(
    initialLocation: '/',
    debugLogDiagnostics: true,
    redirect: (context, state) {
      final auth = authState.valueOrNull;
      final isLoggedIn = auth?.isAuthenticated ?? false;
      final currentPath = state.matchedLocation;

      // Splash always allowed
      if (currentPath == '/') return null;

      // Family view doesn't require auth (token-based)
      if (currentPath.startsWith('/family/')) return null;

      // If not logged in, only allow auth-related routes
      if (!isLoggedIn) {
        if (currentPath == '/onboarding' || currentPath.startsWith('/login')) {
          return null;
        }
        return '/login';
      }

      // If logged in, redirect away from auth screens
      if (currentPath == '/login' ||
          currentPath == '/login/otp' ||
          currentPath == '/onboarding') {
        return '/home/dashboard';
      }

      return null;
    },
    routes: [
      GoRoute(
        path: '/',
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(
        path: '/onboarding',
        builder: (context, state) => const OnboardingScreen(),
      ),
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
        routes: [
          GoRoute(
            path: 'otp',
            builder: (context, state) => const OtpScreen(),
          ),
        ],
      ),
      GoRoute(
        path: '/abha-verify',
        builder: (context, state) => const AbhaVerificationScreen(),
      ),
      GoRoute(
        path: '/family/:token',
        builder: (context, state) => FamilyViewScreen(
          token: state.pathParameters['token']!,
        ),
      ),
      ShellRoute(
        builder: (context, state, child) => MainShell(child: child),
        routes: [
          GoRoute(
            path: '/home/dashboard',
            pageBuilder: (context, state) => const NoTransitionPage(
              child: HomeTab(),
            ),
          ),
          GoRoute(
            path: '/home/progress',
            pageBuilder: (context, state) => const NoTransitionPage(
              child: ProgressTab(),
            ),
          ),
          GoRoute(
            path: '/home/my-day',
            pageBuilder: (context, state) => const NoTransitionPage(
              child: MyDayTab(),
            ),
          ),
          GoRoute(
            path: '/home/learn',
            pageBuilder: (context, state) => const NoTransitionPage(
              child: LearnTab(),
            ),
          ),
        ],
      ),
    ],
  );
});
