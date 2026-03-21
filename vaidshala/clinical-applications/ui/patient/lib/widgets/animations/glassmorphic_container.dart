import 'dart:ui';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import '../../theme/motion.dart';

/// Frosted glass container for bottom sheets and overlays.
/// On web: falls back to semi-transparent solid at 0.85 opacity (no BackdropFilter).
/// On native: full blur effect with [AppMotion.kGlassOpacity] overlay.
class GlassmorphicContainer extends StatelessWidget {
  final Widget child;
  final double borderRadius;
  final double blurSigma;
  final Color borderColor;

  const GlassmorphicContainer({
    super.key,
    required this.child,
    this.borderRadius = AppMotion.kSheetBorderRadius,
    this.blurSigma = AppMotion.kGlassBlurSigma,
    this.borderColor = Colors.white30,
  });

  @override
  Widget build(BuildContext context) {
    final shape = RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(borderRadius)),
      side: BorderSide(color: borderColor, width: 1),
    );

    if (kIsWeb) {
      // Web fallback: solid semi-transparent container
      return Container(
        decoration: ShapeDecoration(
          color: Colors.white.withOpacity(0.85),
          shape: shape,
        ),
        child: child,
      );
    }

    // Native: full blur
    return ClipRRect(
      borderRadius: BorderRadius.vertical(top: Radius.circular(borderRadius)),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: blurSigma, sigmaY: blurSigma),
        child: Container(
          decoration: ShapeDecoration(
            color: Colors.white.withOpacity(AppMotion.kGlassOpacity),
            shape: shape,
          ),
          child: child,
        ),
      ),
    );
  }
}
