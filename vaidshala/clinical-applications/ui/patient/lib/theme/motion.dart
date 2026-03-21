import 'package:flutter/physics.dart';
import 'package:flutter/animation.dart';

/// Central motion constants for CRED-style animations.
/// Every animation in the app references these — never hardcode durations or curves.
class AppMotion {
  AppMotion._();

  // Spring physics
  static const kCreditSpring = SpringDescription(mass: 1, stiffness: 300, damping: 22);
  static const double kSpringScaleMin = 0.96;

  // Curves
  static const Curve kDecelerate = Cubic(0.25, 0.46, 0.45, 0.94);

  // Durations
  static const Duration kStaggerDelay = Duration(milliseconds: 80);
  static const Duration kEntranceDuration = Duration(milliseconds: 400);
  static const Duration kCountUpDuration = Duration(milliseconds: 800);
  static const Duration kProgressFillDuration = Duration(milliseconds: 600);
  static const Duration kPageTransitionDuration = Duration(milliseconds: 300);
  static const Duration kPulseDuration = Duration(milliseconds: 2000);

  // Sizes
  static const double kSlideOffset = 30.0;
  static const double kGlassBlurSigma = 20.0;
  static const double kGlassOpacity = 0.15;
  static const double kSheetBorderRadius = 24.0;
  static const double kPulseScale = 1.05;

  // Limits
  static const int kMaxStaggerItems = 8;
}
