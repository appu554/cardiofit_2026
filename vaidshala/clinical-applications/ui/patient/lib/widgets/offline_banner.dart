import 'package:flutter/material.dart';


class OfflineBanner extends StatelessWidget {
  const OfflineBanner({super.key});

  @override
  Widget build(BuildContext context) {
    // For now, always online — will integrate connectivity_plus later
    return const SizedBox.shrink();
  }
}
