import 'package:flutter/material.dart';
import '../../theme/motion.dart';
import 'staggered_item.dart';

/// Drop-in replacement for Column that auto-wraps each child in StaggeredItem.
class StaggeredColumn extends StatelessWidget {
  final List<Widget> children;
  final MainAxisAlignment mainAxisAlignment;
  final CrossAxisAlignment crossAxisAlignment;
  final MainAxisSize mainAxisSize;
  final Duration staggerDelay;

  const StaggeredColumn({
    super.key,
    required this.children,
    this.mainAxisAlignment = MainAxisAlignment.start,
    this.crossAxisAlignment = CrossAxisAlignment.start,
    this.mainAxisSize = MainAxisSize.min,
    this.staggerDelay = AppMotion.kStaggerDelay,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisAlignment: mainAxisAlignment,
      crossAxisAlignment: crossAxisAlignment,
      mainAxisSize: mainAxisSize,
      children: [
        for (final entry in children.asMap().entries)
          StaggeredItem(
            index: entry.key,
            delay: staggerDelay,
            child: entry.value,
          ),
      ],
    );
  }
}
