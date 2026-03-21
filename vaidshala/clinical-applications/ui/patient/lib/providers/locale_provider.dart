// lib/providers/locale_provider.dart
import 'dart:ui';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'settings_provider.dart';

final localeProvider = Provider<Locale>((ref) {
  final settings = ref.watch(settingsProvider);
  return Locale(settings.language);
});
