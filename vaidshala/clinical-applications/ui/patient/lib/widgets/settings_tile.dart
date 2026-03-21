// lib/widgets/settings_tile.dart
import 'package:flutter/material.dart';
import '../theme.dart';
import 'animations/animations.dart';

class SettingsTile extends StatelessWidget {
  final IconData icon;
  final String title;
  final Widget? trailing;
  final VoidCallback? onTap;

  const SettingsTile({
    super.key,
    required this.icon,
    required this.title,
    this.trailing,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final tile = ListTile(
      leading: Icon(icon, color: AppColors.primaryTeal),
      title: Text(title),
      trailing: trailing ?? (onTap != null ? const Icon(Icons.chevron_right) : null),
      onTap: onTap != null ? null : null,
    );

    if (onTap != null) {
      return SpringTapCard(onTap: onTap!, child: tile);
    }
    return tile;
  }
}
