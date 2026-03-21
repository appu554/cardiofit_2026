// lib/widgets/family_share_button.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class FamilyShareButton extends StatefulWidget {
  final VoidCallback? onShare;

  const FamilyShareButton({super.key, this.onShare});

  @override
  State<FamilyShareButton> createState() => _FamilyShareButtonState();
}

class _FamilyShareButtonState extends State<FamilyShareButton> {
  String? _shareLink;

  void _generateLink() {
    setState(() {
      _shareLink = 'https://vaidshala.health/family/rajesh-kumar-abc123';
    });
    widget.onShare?.call();
  }

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: const Icon(Icons.family_restroom, color: AppColors.primaryTeal),
      title: const Text('Share Health Plan'),
      subtitle: _shareLink != null
          ? Text(_shareLink!, style: const TextStyle(fontSize: 12))
          : null,
      trailing: FilledButton(
        onPressed: _shareLink == null ? _generateLink : null,
        child: Text(_shareLink == null ? 'Generate Link' : 'Shared'),
      ),
    );
  }
}
