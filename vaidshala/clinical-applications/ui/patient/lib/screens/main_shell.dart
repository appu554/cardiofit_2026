// lib/screens/main_shell.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/notifications_provider.dart';
import '../theme/motion.dart';
import '../widgets/animations/animations.dart';
import '../widgets/offline_banner.dart';

class MainShell extends ConsumerStatefulWidget {
  final Widget child;

  const MainShell({super.key, required this.child});

  static const _tabs = [
    '/home/dashboard',
    '/home/progress',
    '/home/my-day',
    '/home/learn',
  ];

  @override
  ConsumerState<MainShell> createState() => _MainShellState();
}

class _MainShellState extends ConsumerState<MainShell> {
  double _appBarElevation = 0;

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    final idx = MainShell._tabs.indexWhere((t) => location.startsWith(t));
    return idx >= 0 ? idx : 0;
  }

  NavigationDestination _buildNavDestination({
    required IconData icon,
    required IconData selectedIcon,
    required String label,
  }) {
    return NavigationDestination(
      icon: Icon(icon),
      selectedIcon: TweenAnimationBuilder<double>(
        tween: Tween(begin: 1.0, end: 1.2),
        duration: AppMotion.kEntranceDuration,
        curve: AppMotion.kDecelerate,
        builder: (context, scale, child) => Transform.scale(
          scale: scale,
          child: child,
        ),
        child: Icon(selectedIcon),
      ),
      label: label,
    );
  }

  @override
  Widget build(BuildContext context) {
    final currentIndex = _currentIndex(context);
    final unreadCount = ref.watch(unreadCountProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.person_outline),
          onPressed: () => context.push('/settings'),
        ),
        title: Text(
          'Vaidshala',
          style: Theme.of(context).textTheme.titleLarge,
        ),
        centerTitle: true,
        elevation: _appBarElevation,
        actions: [
          Stack(
            alignment: Alignment.center,
            children: [
              IconButton(
                icon: const Icon(Icons.notifications_outlined),
                onPressed: () => context.push('/notifications'),
              ),
              if (unreadCount > 0)
                Positioned(
                  right: 8,
                  top: 8,
                  child: PulsingWidget(
                    child: Container(
                      padding: const EdgeInsets.all(4),
                      decoration: const BoxDecoration(
                        color: Colors.red,
                        shape: BoxShape.circle,
                      ),
                      constraints:
                          const BoxConstraints(minWidth: 16, minHeight: 16),
                      child: Text(
                        '$unreadCount',
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                        ),
                        textAlign: TextAlign.center,
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
      body: NotificationListener<ScrollNotification>(
        onNotification: (notification) {
          final newElevation = notification.metrics.pixels > 0 ? 2.0 : 0.0;
          if (newElevation != _appBarElevation) {
            setState(() => _appBarElevation = newElevation);
          }
          return false;
        },
        child: Column(
          children: [
            const OfflineBanner(),
            Expanded(child: widget.child),
          ],
        ),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: currentIndex,
        onDestinationSelected: (i) => context.go(MainShell._tabs[i]),
        destinations: [
          _buildNavDestination(
            icon: Icons.home_outlined,
            selectedIcon: Icons.home,
            label: 'Home',
          ),
          _buildNavDestination(
            icon: Icons.trending_up_outlined,
            selectedIcon: Icons.trending_up,
            label: 'Progress',
          ),
          _buildNavDestination(
            icon: Icons.today_outlined,
            selectedIcon: Icons.today,
            label: 'My Day',
          ),
          _buildNavDestination(
            icon: Icons.school_outlined,
            selectedIcon: Icons.school,
            label: 'Learn',
          ),
        ],
      ),
    );
  }
}
