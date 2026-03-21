// lib/widgets/language_selector.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class LanguageSelector extends StatelessWidget {
  final String current;
  final ValueChanged<String> onChanged;

  const LanguageSelector({
    super.key,
    required this.current,
    required this.onChanged,
  });

  static const _languages = {
    'en': 'English',
    'hi': 'Hindi',
    'ta': 'Tamil',
    'te': 'Telugu',
    'kn': 'Kannada',
    'ml': 'Malayalam',
    'bn': 'Bengali',
    'mr': 'Marathi',
  };

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: const Icon(Icons.language, color: AppColors.primaryTeal),
      title: const Text('Language'),
      trailing: DropdownButton<String>(
        value: current,
        underline: const SizedBox.shrink(),
        items: _languages.entries
            .map((e) => DropdownMenuItem(value: e.key, child: Text(e.value)))
            .toList(),
        onChanged: (v) {
          if (v != null) onChanged(v);
        },
      ),
    );
  }
}
