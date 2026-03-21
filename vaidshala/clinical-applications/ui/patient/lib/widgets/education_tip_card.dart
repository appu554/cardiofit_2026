import 'package:flutter/material.dart';
import '../theme.dart';

class EducationTipCard extends StatefulWidget {
  final String tip;
  final IconData icon;

  const EducationTipCard({
    super.key,
    required this.tip,
    this.icon = Icons.lightbulb_outline,
  });

  @override
  State<EducationTipCard> createState() => _EducationTipCardState();
}

class _EducationTipCardState extends State<EducationTipCard> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final sentences = widget.tip.split('. ');
    final title = sentences.first;
    final body = sentences.length > 1 ? sentences.sublist(1).join('. ') : '';

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: InkWell(
        onTap: body.isEmpty ? null : () => setState(() => _expanded = !_expanded),
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(widget.icon, size: 20, color: AppColors.primaryTeal),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      title,
                      style: const TextStyle(
                          fontSize: 14, fontWeight: FontWeight.w500),
                    ),
                  ),
                  if (body.isNotEmpty)
                    AnimatedRotation(
                      turns: _expanded ? 0.5 : 0,
                      duration: const Duration(milliseconds: 200),
                      child: const Icon(Icons.expand_more,
                          color: AppColors.textSecondary),
                    ),
                ],
              ),
              AnimatedCrossFade(
                firstChild: const SizedBox.shrink(),
                secondChild: Padding(
                  padding: const EdgeInsets.only(top: 8, left: 30),
                  child: Text(
                    body,
                    style: const TextStyle(
                      fontSize: 13,
                      color: AppColors.textSecondary,
                      height: 1.4,
                    ),
                  ),
                ),
                crossFadeState: _expanded
                    ? CrossFadeState.showSecond
                    : CrossFadeState.showFirst,
                duration: const Duration(milliseconds: 200),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
