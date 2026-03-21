import 'package:flutter/material.dart';
import '../models/clinical_translation.dart';
import '../theme.dart';

class ClinicalTranslationRow extends StatefulWidget {
  final ClinicalTranslation translation;
  const ClinicalTranslationRow({super.key, required this.translation});

  @override
  State<ClinicalTranslationRow> createState() =>
      _ClinicalTranslationRowState();
}

class _ClinicalTranslationRowState extends State<ClinicalTranslationRow> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () => setState(() => _expanded = !_expanded),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: RichText(
                    text: TextSpan(
                      style: DefaultTextStyle.of(context).style,
                      children: [
                        TextSpan(
                          text: widget.translation.clinicalTerm,
                          style: const TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: 14,
                          ),
                        ),
                        const TextSpan(text: '  →  '),
                        TextSpan(
                          text: widget.translation.patientTerm,
                          style: const TextStyle(
                            fontSize: 14,
                            color: AppColors.primaryTeal,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
                AnimatedRotation(
                  turns: _expanded ? 0.5 : 0,
                  duration: const Duration(milliseconds: 200),
                  child: const Icon(Icons.expand_more,
                      size: 20, color: AppColors.textSecondary),
                ),
              ],
            ),
            AnimatedCrossFade(
              firstChild: const SizedBox.shrink(),
              secondChild: Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(
                  widget.translation.explanation,
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
            const Divider(height: 1),
          ],
        ),
      ),
    );
  }
}
