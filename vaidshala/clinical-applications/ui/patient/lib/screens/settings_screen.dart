// lib/screens/settings_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/auth_provider.dart';
import '../providers/settings_provider.dart';
import '../theme.dart';
import '../widgets/family_share_button.dart';
import '../widgets/language_selector.dart';
import '../widgets/animations/animations.dart';
import '../widgets/settings_group.dart';
import '../widgets/settings_tile.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final settings = ref.watch(settingsProvider);
    final authAsync = ref.watch(authStateProvider);
    final auth = authAsync.valueOrNull;

    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          // Account
          StaggeredItem(
            index: 0,
            keepAlive: true,
            child: SettingsGroup(
              title: 'Account',
              children: [
                SettingsTile(
                  icon: Icons.person,
                  title: 'Name',
                  trailing: Text(
                    auth?.patientId != null ? 'Rajesh Kumar' : 'Rajesh Kumar',
                    style: const TextStyle(color: AppColors.textSecondary),
                  ),
                ),
                SettingsTile(
                  icon: Icons.phone,
                  title: 'Phone',
                  trailing: Text(
                    '+91 98765 43210',
                    style: const TextStyle(color: AppColors.textSecondary),
                  ),
                ),
                SettingsTile(
                  icon: Icons.verified_user,
                  title: 'ABHA',
                  trailing: const Text(
                    'Linked — rajesh.kumar@abdm',
                    style: TextStyle(
                      color: AppColors.scoreGreen,
                      fontSize: 12,
                    ),
                  ),
                  onTap: () => context.push('/abha-verify'),
                ),
              ],
            ),
          ),

          // Preferences
          StaggeredItem(
            index: 1,
            keepAlive: true,
            child: SettingsGroup(
              title: 'Preferences',
              children: [
                LanguageSelector(
                  current: settings.language,
                  onChanged: (lang) =>
                      ref.read(settingsProvider.notifier).setLanguage(lang),
                ),
                SettingsTile(
                  icon: Icons.notifications,
                  title: 'Notifications',
                  trailing: Switch(
                    value: settings.notificationsEnabled,
                    onChanged: (_) =>
                        ref.read(settingsProvider.notifier).toggleNotifications(),
                  ),
                ),
              ],
            ),
          ),

          // Family
          StaggeredItem(
            index: 2,
            keepAlive: true,
            child: SettingsGroup(
              title: 'Family',
              children: const [
                FamilyShareButton(),
              ],
            ),
          ),

          // Data
          StaggeredItem(
            index: 3,
            keepAlive: true,
            child: SettingsGroup(
              title: 'Data',
              children: [
                SettingsTile(
                  icon: Icons.download,
                  title: 'Download My Data',
                  onTap: () => ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Coming soon')),
                  ),
                ),
                SettingsTile(
                  icon: Icons.delete_forever,
                  title: 'Delete Account',
                  onTap: () => _showDeleteConfirmation(context),
                ),
              ],
            ),
          ),

          // About
          StaggeredItem(
            index: 4,
            keepAlive: true,
            child: SettingsGroup(
              title: 'About',
              children: const [
                SettingsTile(
                  icon: Icons.info_outline,
                  title: 'App Version',
                  trailing: Text('1.0.0', style: TextStyle(color: AppColors.textSecondary)),
                ),
                SettingsTile(
                  icon: Icons.description,
                  title: 'Terms of Service',
                ),
                SettingsTile(
                  icon: Icons.privacy_tip,
                  title: 'Privacy Policy',
                ),
              ],
            ),
          ),

          // Logout
          StaggeredItem(
            index: 5,
            keepAlive: true,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: SpringTapCard(
                onTap: () {
                  ref.read(authStateProvider.notifier).logout();
                  context.go('/login');
                },
                child: OutlinedButton.icon(
                  onPressed: null,
                  icon: const Icon(Icons.logout, color: AppColors.scoreRed),
                  label: const Text('Log Out',
                      style: TextStyle(color: AppColors.scoreRed)),
                ),
              ),
            ),
          ),

          const SizedBox(height: 32),
        ],
      ),
    );
  }

  void _showDeleteConfirmation(BuildContext context) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Account?'),
        content: const Text(
          'This will permanently delete your account and all health data. '
          'This action cannot be undone.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () {
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('Account deletion requested')),
              );
            },
            style: TextButton.styleFrom(foregroundColor: AppColors.scoreRed),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }
}
