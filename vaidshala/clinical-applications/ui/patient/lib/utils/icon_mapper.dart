import 'package:flutter/material.dart';

/// Maps icon name strings (from API/mock data) to Flutter IconData.
/// Single source of truth — all widgets use this instead of inline maps.
class IconMapper {
  static const _map = {
    'bloodtype': Icons.bloodtype,
    'science': Icons.science,
    'favorite': Icons.favorite,
    'directions_walk': Icons.directions_walk,
    'straighten': Icons.straighten,
    'monitor_heart': Icons.monitor_heart,
    'medication': Icons.medication,
    'restaurant': Icons.restaurant,
    'water_drop': Icons.water_drop,
    'egg': Icons.egg,
    'rice_bowl': Icons.rice_bowl,
    'dinner_dining': Icons.dinner_dining,
    'health_and_safety': Icons.health_and_safety,
    'check_circle_outline': Icons.check_circle_outline,
  };

  static IconData fromString(String name, {IconData fallback = Icons.circle}) =>
      _map[name] ?? fallback;
}
