import 'package:flutter/material.dart';
import '../../theme/motion.dart';

/// Combines fade + pixel-accurate vertical slide. Uses Transform.translate
/// (not SlideTransition) so offset is in pixels, not child-size fractions.
class FadeSlideTransition extends StatelessWidget {
  final Animation<double> animation;
  final double slideOffset;
  final Widget child;

  const FadeSlideTransition({
    super.key,
    required this.animation,
    this.slideOffset = AppMotion.kSlideOffset,
    required this.child,
  });

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: animation,
      builder: (context, child) => Transform.translate(
        offset: Offset(0, slideOffset * (1.0 - animation.value)),
        child: Opacity(
          opacity: animation.value,
          child: child,
        ),
      ),
      child: child,
    );
  }
}
