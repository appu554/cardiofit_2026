# CRED-Level Motion Design Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the Vaidshala Patient App into a CRED-inspired premium experience with spring physics, staggered entrances, glassmorphic sheets, count-up numbers, and Poppins typography — while keeping the clinical light theme.

**Architecture:** Custom animation toolkit (9 reusable widgets + motion constants) built from Flutter primitives. Single new dependency: `google_fonts`. All screens wrapped with staggered entrances, all tappable cards get spring-physics feedback, all numbers animate to values, all bottom sheets get glassmorphic treatment.

**Tech Stack:** Flutter 3.41.5, Dart 3.11, google_fonts ^6.1.0, SpringSimulation, AnimationController, BackdropFilter (web fallback)

**Spec:** `docs/superpowers/specs/2026-03-21-patient-app-cred-motion-design.md`

**Project root:** `vaidshala/clinical-applications/ui/patient/`

**Package name:** `vaidshala_patient` (imports: `package:vaidshala_patient/...`)

---

## File Structure

### New Files (11)
```
lib/theme/motion.dart                              # Motion constants
lib/widgets/animations/animations.dart              # Barrel export
lib/widgets/animations/fade_slide_transition.dart   # Fade + pixel-accurate vertical slide combo
lib/widgets/animations/staggered_item.dart          # Index-delayed entrance animation
lib/widgets/animations/staggered_column.dart        # Auto-staggering Column wrapper
lib/widgets/animations/spring_tap_card.dart         # Spring-physics tap feedback card
lib/widgets/animations/glassmorphic_container.dart  # Frosted glass (web fallback)
lib/widgets/animations/count_up_text.dart           # Number count-up animation
lib/widgets/animations/animated_progress_bar.dart   # Left-to-right fill bar
lib/widgets/animations/pulsing_widget.dart          # Infinite subtle scale pulse
lib/widgets/animations/shake_widget.dart            # Horizontal shake for errors
```

### Modified Files (~22)
```
pubspec.yaml                        # Add google_fonts
lib/theme.dart                      # Poppins type scale
lib/router.dart                     # Fade-through page transitions
lib/screens/main_shell.dart         # Shell animations (nav, badge, appbar)
lib/screens/home_tab.dart           # Staggered + spring + count-up
lib/screens/my_day_tab.dart         # Staggered + spring FAB
lib/screens/progress_tab.dart       # Staggered + count-up + animated bars
lib/screens/learn_tab.dart          # Staggered + spring cards
lib/screens/settings_screen.dart    # Staggered + spring tiles
lib/screens/score_detail_screen.dart # Count-up + animated bars
lib/screens/notifications_screen.dart # Staggered + pulse
lib/widgets/vitals_entry_sheet.dart  # Glassmorphic + staggered
lib/widgets/symptom_logger_sheet.dart # Glassmorphic + staggered chips
lib/widgets/medication_adherence_section.dart # Count-up + staggered
lib/widgets/adherence_ring.dart      # Count-up integration
lib/widgets/domain_breakdown_bar.dart # AnimatedProgressBar swap
lib/widgets/notification_item.dart   # Swipe direction fix + pulse
lib/widgets/bp_entry_card.dart       # ShakeWidget on validation
lib/widgets/glucose_entry_card.dart  # ShakeWidget on validation
lib/widgets/weight_entry_card.dart   # ShakeWidget on validation
lib/widgets/settings_tile.dart       # SpringTapCard wrap
lib/widgets/settings_group.dart      # StaggeredItem wrap
lib/widgets/notification_date_group.dart # FadeSlideTransition on date headers
lib/widgets/full_sparkline_chart.dart # Animated line draw
```

---

### Task 1: Add google_fonts dependency + motion constants

**Files:**
- Modify: `pubspec.yaml`
- Create: `lib/theme/motion.dart`

- [ ] **Step 1: Add google_fonts to pubspec.yaml**

In `pubspec.yaml`, add `google_fonts: ^6.1.0` under the `# UI` section after `smooth_page_indicator`:

```yaml
  # UI
  smooth_page_indicator: ^1.2.0+3
  google_fonts: ^6.1.0
```

- [ ] **Step 2: Create motion constants file**

Create `lib/theme/motion.dart`:

```dart
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
```

- [ ] **Step 3: Run pub get**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter pub get`
Expected: resolves google_fonts, exits 0

- [ ] **Step 4: Commit**

```bash
git add pubspec.yaml lib/theme/motion.dart
git commit -m "feat: add google_fonts dep and motion constants for CRED animations"
```

---

### Task 2: FadeSlideTransition + StaggeredItem + StaggeredColumn

**Files:**
- Create: `lib/widgets/animations/fade_slide_transition.dart`
- Create: `lib/widgets/animations/staggered_item.dart`
- Create: `lib/widgets/animations/staggered_column.dart`

- [ ] **Step 1: Create FadeSlideTransition**

Create `lib/widgets/animations/fade_slide_transition.dart`:

```dart
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
```

- [ ] **Step 2: Create StaggeredItem**

Create `lib/widgets/animations/staggered_item.dart`:

```dart
import 'package:flutter/material.dart';
import '../../theme/motion.dart';
import 'fade_slide_transition.dart';

/// Wraps a child with a staggered entrance animation.
/// Each item delays by [index * delay] before animating in.
class StaggeredItem extends StatefulWidget {
  final int index;
  final Widget child;
  final Duration duration;
  final Duration delay;
  final bool keepAlive;

  const StaggeredItem({
    super.key,
    required this.index,
    required this.child,
    this.duration = AppMotion.kEntranceDuration,
    this.delay = AppMotion.kStaggerDelay,
    this.keepAlive = false,
  });

  @override
  State<StaggeredItem> createState() => _StaggeredItemState();
}

class _StaggeredItemState extends State<StaggeredItem>
    with SingleTickerProviderStateMixin, AutomaticKeepAliveClientMixin {
  late final AnimationController _controller;
  late final Animation<double> _animation;

  @override
  bool get wantKeepAlive => widget.keepAlive;

  @override
  void initState() {
    super.initState();

    // Clamp index to max stagger items
    final effectiveIndex = widget.index.clamp(0, AppMotion.kMaxStaggerItems);
    final totalDelay = widget.delay * effectiveIndex;
    final totalDuration = widget.duration + totalDelay;

    _controller = AnimationController(
      vsync: this,
      duration: totalDuration,
    );

    // If beyond max items, show instantly (no animation)
    if (widget.index > AppMotion.kMaxStaggerItems) {
      _animation = const AlwaysStoppedAnimation(1.0);
      return;
    }

    final startFraction =
        totalDuration.inMilliseconds > 0
            ? totalDelay.inMilliseconds / totalDuration.inMilliseconds
            : 0.0;

    _animation = CurvedAnimation(
      parent: _controller,
      curve: Interval(startFraction, 1.0, curve: AppMotion.kDecelerate),
    );

    _controller.forward();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    super.build(context); // Required for AutomaticKeepAliveClientMixin
    return FadeSlideTransition(
      animation: _animation,
      child: widget.child,
    );
  }
}
```

- [ ] **Step 3: Create StaggeredColumn**

Create `lib/widgets/animations/staggered_column.dart`:

```dart
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
```

- [ ] **Step 4: Commit**

```bash
git add lib/widgets/animations/fade_slide_transition.dart lib/widgets/animations/staggered_item.dart lib/widgets/animations/staggered_column.dart
git commit -m "feat: add FadeSlideTransition, StaggeredItem, and StaggeredColumn widgets"
```

---

### Task 3: SpringTapCard + GlassmorphicContainer

**Files:**
- Create: `lib/widgets/animations/spring_tap_card.dart`
- Create: `lib/widgets/animations/glassmorphic_container.dart`

- [ ] **Step 1: Create SpringTapCard**

Create `lib/widgets/animations/spring_tap_card.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:flutter/physics.dart';
import '../../theme/motion.dart';

/// Card that responds to tap with spring-physics scale bounce.
/// Uses a single AnimationController with animateWith() for seamless interruption.
class SpringTapCard extends StatefulWidget {
  final Widget child;
  final VoidCallback? onTap;
  final double borderRadius;
  final double elevation;

  const SpringTapCard({
    super.key,
    required this.child,
    this.onTap,
    this.borderRadius = 12,
    this.elevation = 1,
  });

  @override
  State<SpringTapCard> createState() => _SpringTapCardState();
}

class _SpringTapCardState extends State<SpringTapCard>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController.unbounded(
      vsync: this,
      value: 1.0,
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _onTapDown(TapDownDetails _) {
    _controller.animateWith(
      SpringSimulation(
        AppMotion.kCreditSpring,
        _controller.value,
        AppMotion.kSpringScaleMin,
        _controller.velocity,
      ),
    );
  }

  void _onTapUp(TapUpDetails _) {
    _controller.animateWith(
      SpringSimulation(
        AppMotion.kCreditSpring,
        _controller.value,
        1.0,
        _controller.velocity,
      ),
    );
  }

  void _onTapCancel() {
    _controller.animateWith(
      SpringSimulation(
        AppMotion.kCreditSpring,
        _controller.value,
        1.0,
        _controller.velocity,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTapDown: _onTapDown,
      onTapUp: _onTapUp,
      onTapCancel: _onTapCancel,
      onTap: widget.onTap,
      child: AnimatedBuilder(
        animation: _controller,
        builder: (context, child) => Transform.scale(
          scale: _controller.value,
          child: child,
        ),
        child: Material(
          elevation: widget.elevation,
          borderRadius: BorderRadius.circular(widget.borderRadius),
          clipBehavior: Clip.antiAlias,
          child: widget.child,
        ),
      ),
    );
  }
}
```

- [ ] **Step 2: Create GlassmorphicContainer**

Create `lib/widgets/animations/glassmorphic_container.dart`:

```dart
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
```

- [ ] **Step 3: Commit**

```bash
git add lib/widgets/animations/spring_tap_card.dart lib/widgets/animations/glassmorphic_container.dart
git commit -m "feat: add SpringTapCard with spring physics and GlassmorphicContainer"
```

---

### Task 4: CountUpText + AnimatedProgressBar + PulsingWidget + ShakeWidget + barrel

**Files:**
- Create: `lib/widgets/animations/count_up_text.dart`
- Create: `lib/widgets/animations/animated_progress_bar.dart`
- Create: `lib/widgets/animations/pulsing_widget.dart`
- Create: `lib/widgets/animations/shake_widget.dart`
- Create: `lib/widgets/animations/animations.dart`

- [ ] **Step 1: Create CountUpText**

Create `lib/widgets/animations/count_up_text.dart`:

```dart
import 'package:flutter/material.dart';
import '../../theme/motion.dart';

/// Animates a number from 0 → target on first build.
/// Re-animates when value changes.
class CountUpText extends StatefulWidget {
  final double value;
  final Duration duration;
  final TextStyle? style;
  final String suffix;
  final int decimalPlaces;

  const CountUpText({
    super.key,
    required this.value,
    this.duration = AppMotion.kCountUpDuration,
    this.style,
    this.suffix = '',
    this.decimalPlaces = 0,
  });

  @override
  State<CountUpText> createState() => _CountUpTextState();
}

class _CountUpTextState extends State<CountUpText>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _animation;
  CurvedAnimation? _curved;
  double _previousValue = 0;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: widget.duration,
    );
    _curved = CurvedAnimation(parent: _controller, curve: AppMotion.kDecelerate);
    _animation = Tween<double>(begin: 0, end: widget.value).animate(_curved!);
    _controller.forward();
  }

  @override
  void didUpdateWidget(CountUpText oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.value != widget.value) {
      _previousValue = oldWidget.value;
      _curved?.dispose();
      _curved = CurvedAnimation(parent: _controller, curve: AppMotion.kDecelerate);
      _animation = Tween<double>(begin: _previousValue, end: widget.value)
          .animate(_curved!);
      _controller
        ..reset()
        ..forward();
    }
  }

  @override
  void dispose() {
    _curved?.dispose();
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _animation,
      builder: (context, _) {
        final text = widget.decimalPlaces > 0
            ? _animation.value.toStringAsFixed(widget.decimalPlaces)
            : _animation.value.round().toString();
        return Text(
          '$text${widget.suffix}',
          style: widget.style,
        );
      },
    );
  }
}
```

- [ ] **Step 2: Create AnimatedProgressBar**

Create `lib/widgets/animations/animated_progress_bar.dart`:

```dart
import 'package:flutter/material.dart';
import '../../theme/motion.dart';

/// Progress bar that fills left→right on first build.
class AnimatedProgressBar extends StatefulWidget {
  final double value;
  final Duration duration;
  final Color? color;
  final Color? backgroundColor;
  final Gradient? gradient;
  final double height;
  final double borderRadius;

  const AnimatedProgressBar({
    super.key,
    required this.value,
    this.duration = AppMotion.kProgressFillDuration,
    this.color,
    this.backgroundColor,
    this.gradient,
    this.height = 6,
    this.borderRadius = 3,
  });

  @override
  State<AnimatedProgressBar> createState() => _AnimatedProgressBarState();
}

class _AnimatedProgressBarState extends State<AnimatedProgressBar>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _animation;
  CurvedAnimation? _curved;
  double _previousValue = 0;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: widget.duration,
    );
    _curved = CurvedAnimation(parent: _controller, curve: AppMotion.kDecelerate);
    _animation = Tween<double>(begin: 0, end: widget.value.clamp(0, 1))
        .animate(_curved!);
    _controller.forward();
  }

  @override
  void didUpdateWidget(AnimatedProgressBar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.value != widget.value) {
      _previousValue = oldWidget.value.clamp(0, 1);
      _curved?.dispose();
      _curved = CurvedAnimation(parent: _controller, curve: AppMotion.kDecelerate);
      _animation = Tween<double>(
        begin: _previousValue,
        end: widget.value.clamp(0, 1),
      ).animate(_curved!);
      _controller
        ..reset()
        ..forward();
    }
  }

  @override
  void dispose() {
    _curved?.dispose();
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final bgColor = widget.backgroundColor ?? Colors.grey.shade200;
    final fgColor = widget.color ?? Theme.of(context).colorScheme.primary;

    return AnimatedBuilder(
      animation: _animation,
      builder: (context, _) {
        return SizedBox(
          height: widget.height,
          child: ClipRRect(
            borderRadius: BorderRadius.circular(widget.borderRadius),
            child: Stack(
              children: [
                Container(color: bgColor),
                FractionallySizedBox(
                  widthFactor: _animation.value,
                  child: Container(
                    decoration: BoxDecoration(
                      color: widget.gradient == null ? fgColor : null,
                      gradient: widget.gradient,
                    ),
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}
```

- [ ] **Step 3: Create PulsingWidget**

Create `lib/widgets/animations/pulsing_widget.dart`:

```dart
import 'package:flutter/material.dart';
import '../../theme/motion.dart';

/// Infinite subtle scale pulse for attention-grabbing elements.
class PulsingWidget extends StatefulWidget {
  final Widget child;
  final Duration duration;
  final double minScale;
  final double maxScale;

  const PulsingWidget({
    super.key,
    required this.child,
    this.duration = AppMotion.kPulseDuration,
    this.minScale = 1.0,
    this.maxScale = AppMotion.kPulseScale,
  });

  @override
  State<PulsingWidget> createState() => _PulsingWidgetState();
}

class _PulsingWidgetState extends State<PulsingWidget>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _scaleAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: widget.duration,
    )..repeat(reverse: true);
    _scaleAnimation = Tween<double>(
      begin: widget.minScale,
      end: widget.maxScale,
    ).animate(CurvedAnimation(parent: _controller, curve: Curves.easeInOut));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ScaleTransition(
      scale: _scaleAnimation,
      child: widget.child,
    );
  }
}
```

- [ ] **Step 4: Create ShakeWidget**

Create `lib/widgets/animations/shake_widget.dart`:

```dart
import 'dart:math' as math;
import 'package:flutter/material.dart';

/// Horizontal shake for validation errors.
/// Call [shake()] via GlobalKey<ShakeWidgetState> to trigger.
class ShakeWidget extends StatefulWidget {
  final Widget child;
  final int shakeCount;
  final double shakeOffset;

  const ShakeWidget({
    super.key,
    required this.child,
    this.shakeCount = 3,
    this.shakeOffset = 6,
  });

  @override
  State<ShakeWidget> createState() => ShakeWidgetState();
}

class ShakeWidgetState extends State<ShakeWidget>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 400),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void shake() {
    _controller
      ..reset()
      ..forward();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, child) {
        final dx = math.sin(_controller.value * math.pi * widget.shakeCount) *
            widget.shakeOffset;
        return Transform.translate(
          offset: Offset(dx, 0),
          child: child,
        );
      },
      child: widget.child,
    );
  }
}
```

- [ ] **Step 5: Create barrel export file**

Create `lib/widgets/animations/animations.dart`:

```dart
export 'fade_slide_transition.dart';
export 'staggered_item.dart';
export 'staggered_column.dart';
export 'spring_tap_card.dart';
export 'glassmorphic_container.dart';
export 'count_up_text.dart';
export 'animated_progress_bar.dart';
export 'pulsing_widget.dart';
export 'shake_widget.dart';
```

- [ ] **Step 6: Commit**

```bash
git add lib/widgets/animations/
git commit -m "feat: add CountUpText, AnimatedProgressBar, PulsingWidget, ShakeWidget, and barrel file"
```

---

### Task 5: Animation widget unit tests

**Files:**
- Create: `test/widgets/animations/fade_slide_transition_test.dart`
- Create: `test/widgets/animations/spring_tap_card_test.dart`
- Create: `test/widgets/animations/staggered_item_test.dart`
- Create: `test/widgets/animations/glassmorphic_container_test.dart`
- Create: `test/widgets/animations/count_up_text_test.dart`

- [ ] **Step 1: Test FadeSlideTransition controller lifecycle**

Create `test/widgets/animations/fade_slide_transition_test.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/fade_slide_transition.dart';

void main() {
  testWidgets('FadeSlideTransition slides from offset to zero', (tester) async {
    final controller = AnimationController(
      vsync: const TestVSync(),
      duration: const Duration(milliseconds: 400),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: FadeSlideTransition(
          animation: controller,
          child: const Text('Hello'),
        ),
      ),
    );

    // Initially at offset (opacity 0)
    expect(find.text('Hello'), findsOneWidget);

    controller.forward();
    await tester.pump(const Duration(milliseconds: 200)); // halfway
    // Should be partially visible

    await tester.pump(const Duration(milliseconds: 200)); // complete
    controller.dispose();
  });
}
```

- [ ] **Step 2: Test SpringTapCard spring physics**

Create `test/widgets/animations/spring_tap_card_test.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/spring_tap_card.dart';

void main() {
  // Use pump() with duration, NOT pumpAndSettle() — springs don't "settle" in test timeouts
  const kTestSettleDuration = Duration(seconds: 2);

  testWidgets('SpringTapCard scales down on press and back on release', (tester) async {
    bool tapped = false;

    await tester.pumpWidget(
      MaterialApp(
        home: SpringTapCard(
          onTap: () => tapped = true,
          child: const SizedBox(width: 100, height: 100),
        ),
      ),
    );

    // Press down
    final gesture = await tester.press(find.byType(SpringTapCard));
    await tester.pump(const Duration(milliseconds: 100));
    // Scale should be < 1.0 (spring moving toward 0.96)

    // Release
    await gesture.up();
    await tester.pump(kTestSettleDuration);
    // Scale should be back to 1.0
  });
}
```

- [ ] **Step 3: Test StaggeredItem interval fractions**

Create `test/widgets/animations/staggered_item_test.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/staggered_item.dart';

void main() {
  testWidgets('StaggeredItem with index 0 starts immediately', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: StaggeredItem(
          index: 0,
          child: Text('Item 0'),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('Item 0'), findsOneWidget);
  });

  testWidgets('StaggeredItem beyond kMaxStaggerItems shows instantly', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: StaggeredItem(
          index: 100,
          child: Text('Item 100'),
        ),
      ),
    );
    // Should be visible immediately (no animation)
    expect(find.text('Item 100'), findsOneWidget);
  });
}
```

- [ ] **Step 4: Test GlassmorphicContainer web fallback**

Create `test/widgets/animations/glassmorphic_container_test.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/glassmorphic_container.dart';

void main() {
  testWidgets('GlassmorphicContainer renders child', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: GlassmorphicContainer(
          child: Text('Sheet content'),
        ),
      ),
    );
    expect(find.text('Sheet content'), findsOneWidget);
  });
}
```

- [ ] **Step 5: Test CountUpText re-animation**

Create `test/widgets/animations/count_up_text_test.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/widgets/animations/count_up_text.dart';

void main() {
  testWidgets('CountUpText animates from 0 to value', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: CountUpText(value: 75, suffix: '%'),
      ),
    );

    // Initially at 0
    expect(find.text('0%'), findsOneWidget);

    // After animation completes
    await tester.pump(const Duration(milliseconds: 800));
    expect(find.text('75%'), findsOneWidget);
  });
}
```

- [ ] **Step 6: Run tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test test/widgets/animations/`
Expected: All tests pass

- [ ] **Step 7: Commit**

```bash
git add test/widgets/animations/
git commit -m "test: add unit tests for animation toolkit widgets"
```

---

### Task 6: Poppins typography + theme update

**Files:**
- Modify: `lib/theme.dart`

- [ ] **Step 1: Update theme.dart with Poppins type scale**

Replace the entire `lib/theme.dart` with:

```dart
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

class AppColors {
  // Score ring zones
  static const Color scoreGreen = Color(0xFF2E7D32);
  static const Color scoreYellow = Color(0xFFF9A825);
  static const Color scoreRed = Color(0xFFC62828);

  // Tenant-overridable defaults
  static const Color primaryNavy = Color(0xFF1B3A5C);
  static const Color primaryTeal = Color(0xFF00897B);
  static const Color surfaceLight = Color(0xFFF5F7FA);
  static const Color textPrimary = Color(0xFF212121);
  static const Color textSecondary = Color(0xFF757575);

  // Functional
  static const Color coachingGreen = Color(0xFFE8F5E9);
  static const Color alertAmber = Color(0xFFFFF8E1);
  static const Color offlineBanner = Color(0xFFFFA726);

  static Color scoreColor(int score) {
    if (score >= 60) return scoreGreen;
    if (score >= 40) return scoreYellow;
    return scoreRed;
  }
}

ThemeData buildAppTheme({Color? primaryColor}) {
  final primary = primaryColor ?? AppColors.primaryTeal;
  final poppins = GoogleFonts.poppinsTextTheme();

  return ThemeData(
    useMaterial3: true,
    colorSchemeSeed: primary,
    brightness: Brightness.light,
    scaffoldBackgroundColor: AppColors.surfaceLight,
    appBarTheme: const AppBarTheme(
      backgroundColor: Colors.white,
      foregroundColor: AppColors.textPrimary,
      elevation: 0,
    ),
    cardTheme: CardThemeData(
      elevation: 1,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
      ),
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
    ),
    textTheme: poppins.copyWith(
      displayLarge: poppins.displayLarge?.copyWith(
        fontSize: 32, fontWeight: FontWeight.w700,
        color: AppColors.textPrimary, letterSpacing: -0.5,
      ),
      headlineLarge: poppins.headlineLarge?.copyWith(
        fontSize: 28, fontWeight: FontWeight.w700,
        color: AppColors.textPrimary, letterSpacing: -0.3,
      ),
      headlineMedium: poppins.headlineMedium?.copyWith(
        fontSize: 24, fontWeight: FontWeight.w600,
        color: AppColors.textPrimary, letterSpacing: -0.2,
      ),
      titleLarge: poppins.titleLarge?.copyWith(
        fontSize: 20, fontWeight: FontWeight.w600,
        color: AppColors.textPrimary, letterSpacing: 0,
      ),
      titleMedium: poppins.titleMedium?.copyWith(
        fontSize: 16, fontWeight: FontWeight.w600,
        color: AppColors.textPrimary, letterSpacing: 0.1,
      ),
      bodyLarge: poppins.bodyLarge?.copyWith(
        fontSize: 16, fontWeight: FontWeight.w400,
        color: AppColors.textPrimary, letterSpacing: 0.2,
      ),
      bodyMedium: poppins.bodyMedium?.copyWith(
        fontSize: 14, fontWeight: FontWeight.w400,
        color: AppColors.textSecondary, letterSpacing: 0.2,
      ),
      bodySmall: poppins.bodySmall?.copyWith(
        fontSize: 12, fontWeight: FontWeight.w400,
        color: AppColors.textSecondary, letterSpacing: 0.3,
      ),
      labelLarge: poppins.labelLarge?.copyWith(
        fontSize: 14, fontWeight: FontWeight.w500,
        letterSpacing: 0.5,
      ),
      labelSmall: poppins.labelSmall?.copyWith(
        fontSize: 11, fontWeight: FontWeight.w500,
        letterSpacing: 0.5,
      ),
    ),
  );
}
```

- [ ] **Step 2: Verify build compiles**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter build web --no-tree-shake-icons 2>&1 | tail -5`
Expected: exits 0

- [ ] **Step 3: Commit**

```bash
git add lib/theme.dart
git commit -m "feat: switch to Poppins typography with expanded type scale"
```

---

### Task 7: Fade-through page transitions in router

**Files:**
- Modify: `lib/router.dart`

- [ ] **Step 1: Add fade-through transitions to pushed routes**

In `lib/router.dart`, add import at top:

```dart
import 'package:flutter/material.dart';
import 'theme/motion.dart';
```

Replace the three builder-based GoRoutes (`/settings`, `/score-detail`, `/notifications`) with `pageBuilder` using `CustomTransitionPage`:

Replace:
```dart
      GoRoute(
        path: '/settings',
        builder: (context, state) => const SettingsScreen(),
      ),
      GoRoute(
        path: '/score-detail',
        builder: (context, state) => const ScoreDetailScreen(),
      ),
      GoRoute(
        path: '/notifications',
        builder: (context, state) => const NotificationsScreen(),
      ),
```

With:
```dart
      GoRoute(
        path: '/settings',
        pageBuilder: (context, state) => CustomTransitionPage(
          key: state.pageKey,
          child: const SettingsScreen(),
          transitionsBuilder: _fadeThrough,
          transitionDuration: AppMotion.kPageTransitionDuration,
        ),
      ),
      GoRoute(
        path: '/score-detail',
        pageBuilder: (context, state) => CustomTransitionPage(
          key: state.pageKey,
          child: const ScoreDetailScreen(),
          transitionsBuilder: _sharedAxisVertical,
          transitionDuration: AppMotion.kPageTransitionDuration,
        ),
      ),
      GoRoute(
        path: '/notifications',
        pageBuilder: (context, state) => CustomTransitionPage(
          key: state.pageKey,
          child: const NotificationsScreen(),
          transitionsBuilder: _fadeThrough,
          transitionDuration: AppMotion.kPageTransitionDuration,
        ),
      ),
```

Add this helper function at the bottom of the file (outside the provider):
```dart
Widget _fadeThrough(
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child) {
  return FadeTransition(
    opacity: CurvedAnimation(parent: animation, curve: AppMotion.kDecelerate),
    child: child,
  );
}

Widget _sharedAxisVertical(
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child) {
  return AnimatedBuilder(
    animation: animation,
    builder: (context, child) => Transform.translate(
      offset: Offset(0, 20.0 * (1.0 - animation.value)),
      child: Opacity(
        opacity: animation.value,
        child: child,
      ),
    ),
    child: child,
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add lib/router.dart
git commit -m "feat: add fade-through page transitions for pushed routes"
```

---

### Task 8: Main shell animations (nav bar, badge, appbar)

**Files:**
- Modify: `lib/screens/main_shell.dart`

- [ ] **Step 1: Update MainShell with animation imports and pulsing badge**

Replace the entire `lib/screens/main_shell.dart` with:

```dart
// lib/screens/main_shell.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/notifications_provider.dart';
import '../theme/motion.dart';
import '../widgets/animations/animations.dart';
import '../widgets/offline_banner.dart';

class MainShell extends ConsumerStatefulWidget {
  final Widget child;

  const MainShell({super.key, required this.child});

  static const _tabs = [
    '/home/dashboard',
    '/home/progress',
    '/home/my-day',
    '/home/learn',
  ];

  @override
  ConsumerState<MainShell> createState() => _MainShellState();
}

class _MainShellState extends ConsumerState<MainShell> {
  double _appBarElevation = 0;

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    final idx = MainShell._tabs.indexWhere((t) => location.startsWith(t));
    return idx >= 0 ? idx : 0;
  }

  NavigationDestination _buildNavDestination({
    required IconData icon,
    required IconData selectedIcon,
    required String label,
  }) {
    return NavigationDestination(
      icon: Icon(icon),
      selectedIcon: TweenAnimationBuilder<double>(
        tween: Tween(begin: 1.0, end: 1.2),
        duration: AppMotion.kEntranceDuration,
        curve: AppMotion.kDecelerate,
        builder: (context, scale, child) => Transform.scale(
          scale: scale,
          child: child,
        ),
        child: Icon(selectedIcon),
      ),
      label: label,
    );
  }

  @override
  Widget build(BuildContext context) {
    final currentIndex = _currentIndex(context);
    final unreadCount = ref.watch(unreadCountProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.person_outline),
          onPressed: () => context.push('/settings'),
        ),
        title: Text(
          'Vaidshala',
          style: Theme.of(context).textTheme.titleLarge,
        ),
        centerTitle: true,
        elevation: _appBarElevation,
        actions: [
          Stack(
            alignment: Alignment.center,
            children: [
              IconButton(
                icon: const Icon(Icons.notifications_outlined),
                onPressed: () => context.push('/notifications'),
              ),
              if (unreadCount > 0)
                Positioned(
                  right: 8,
                  top: 8,
                  child: PulsingWidget(
                    child: Container(
                      padding: const EdgeInsets.all(4),
                      decoration: const BoxDecoration(
                        color: Colors.red,
                        shape: BoxShape.circle,
                      ),
                      constraints:
                          const BoxConstraints(minWidth: 16, minHeight: 16),
                      child: Text(
                        '$unreadCount',
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                        ),
                        textAlign: TextAlign.center,
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
      body: NotificationListener<ScrollNotification>(
        onNotification: (notification) {
          final newElevation = notification.metrics.pixels > 0 ? 2.0 : 0.0;
          if (newElevation != _appBarElevation) {
            setState(() => _appBarElevation = newElevation);
          }
          return false;
        },
        child: Column(
          children: [
            const OfflineBanner(),
            Expanded(child: widget.child),
          ],
        ),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: currentIndex,
        onDestinationSelected: (i) => context.go(MainShell._tabs[i]),
        destinations: [
          _buildNavDestination(
            icon: Icons.home_outlined,
            selectedIcon: Icons.home,
            label: 'Home',
          ),
          _buildNavDestination(
            icon: Icons.trending_up_outlined,
            selectedIcon: Icons.trending_up,
            label: 'Progress',
          ),
          _buildNavDestination(
            icon: Icons.today_outlined,
            selectedIcon: Icons.today,
            label: 'My Day',
          ),
          _buildNavDestination(
            icon: Icons.school_outlined,
            selectedIcon: Icons.school,
            label: 'Learn',
          ),
        ],
      ),
    );
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add lib/screens/main_shell.dart
git commit -m "feat: add pulsing notification badge and Poppins title to shell"
```

---

### Task 9: Home tab — staggered entrance + spring cards + count-up + animated bars

**Files:**
- Modify: `lib/screens/home_tab.dart`

- [ ] **Step 1: Update HomeTab with animation wrappers**

Replace the entire `lib/screens/home_tab.dart` with:

```dart
// lib/screens/home_tab.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../models/insight.dart';
import '../providers/actions_provider.dart';
import '../providers/drivers_provider.dart';
import '../providers/health_score_provider.dart';
import '../providers/insights_provider.dart';
import '../theme.dart';
import '../theme/motion.dart';
import '../widgets/action_checklist_item.dart';
import '../widgets/animations/animations.dart';
import '../widgets/coaching_card.dart';
import '../widgets/driver_card.dart';
import '../widgets/score_ring.dart';
import '../widgets/skeleton_card.dart';

class HomeTab extends ConsumerWidget {
  const HomeTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scoreAsync = ref.watch(healthScoreProvider);
    final actionsState = ref.watch(actionsProvider);
    final driversAsync = ref.watch(healthDriversProvider);
    final insightsAsync = ref.watch(insightsProvider);

    return SafeArea(
      child: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(healthScoreProvider);
          ref.read(actionsProvider.notifier).refresh();
          ref.invalidate(healthDriversProvider);
          ref.invalidate(insightsProvider);
        },
        child: SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.only(bottom: 80),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Greeting
              StaggeredItem(
                index: 0,
                keepAlive: true,
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    'Namaste, Rajesh',
                    style: Theme.of(context).textTheme.headlineLarge,
                  ),
                ),
              ),

              // Score Ring Card
              StaggeredItem(
                index: 1,
                keepAlive: true,
                child: scoreAsync.when(
                  data: (score) => _ScoreCard(score: score?.score),
                  loading: () => const SkeletonCard(height: 180),
                  error: (_, __) => const _ScoreCard(score: null),
                ),
              ),

              // Coaching Message
              StaggeredItem(
                index: 2,
                keepAlive: true,
                child: insightsAsync.when(
                  data: (insight) {
                    if (insight.coachingMessage == null) {
                      return const SizedBox.shrink();
                    }
                    return Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 4),
                      child: TweenAnimationBuilder<double>(
                        tween: Tween(begin: 0.0, end: 3.0),
                        duration: AppMotion.kEntranceDuration,
                        builder: (context, borderWidth, child) => Container(
                          decoration: BoxDecoration(
                            border: Border(
                              left: BorderSide(
                                color: AppColors.primaryTeal,
                                width: borderWidth,
                              ),
                            ),
                          ),
                          child: child,
                        ),
                        child: CoachingMessageCard(
                          message: insight.coachingMessage!,
                          type: insight.coachingType ?? InsightType.encouragement,
                        ),
                      ),
                    );
                  },
                  loading: () => const SkeletonCard(height: 80),
                  error: (_, __) => const SizedBox.shrink(),
                ),
              ),

              // Today's Actions
              StaggeredItem(
                index: 3,
                keepAlive: true,
                child: _ActionsSection(
                  state: actionsState,
                  onToggle: (id) =>
                      ref.read(actionsProvider.notifier).toggleAction(id),
                ),
              ),

              // Health Drivers
              StaggeredItem(
                index: 4,
                keepAlive: true,
                child: driversAsync.when(
                  data: (drivers) => Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Padding(
                        padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                        child: Text(
                          'Health Drivers',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                      ...drivers.map((d) => HealthDriverCard(driver: d)),
                    ],
                  ),
                  loading: () => const SkeletonCard(height: 100),
                  error: (_, __) => const SizedBox.shrink(),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ScoreCard extends StatelessWidget {
  final int? score;
  const _ScoreCard({this.score});

  @override
  Widget build(BuildContext context) {
    return SpringTapCard(
      onTap: () => context.push('/score-detail'),
      child: Card(
        margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        color: AppColors.primaryNavy,
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Row(
            children: [
              Hero(
                tag: 'score-ring',
                child: ScoreRing(score: score, size: 120),
              ),
              const SizedBox(width: 24),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      'Metabolic Health Score',
                      style: TextStyle(
                        color: Colors.white70,
                        fontSize: 12,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 4),
                    if (score != null)
                      Text(
                        'This month',
                        style: TextStyle(
                          color: Colors.white.withValues(alpha: 0.5),
                          fontSize: 11,
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ActionsSection extends StatelessWidget {
  final ActionsState state;
  final ValueChanged<String> onToggle;

  const _ActionsSection({required this.state, required this.onToggle});

  @override
  Widget build(BuildContext context) {
    if (state.isLoading) {
      return const SkeletonCard(height: 200);
    }

    if (state.actions.isEmpty) {
      return const Card(
        margin: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Center(
            child: Text(
              'Connect to load your health actions',
              style: TextStyle(color: AppColors.textSecondary),
            ),
          ),
        ),
      );
    }

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  "Today's Actions",
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                Text(
                  '${state.completionPct}% complete',
                  style: const TextStyle(
                    fontSize: 12,
                    color: AppColors.textSecondary,
                  ),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: AnimatedProgressBar(
              value: state.completionPct / 100,
              color: AppColors.scoreGreen,
              height: 6,
            ),
          ),
          const SizedBox(height: 8),
          ...state.actions.map(
            (action) => ActionChecklistItem(
              action: action,
              onToggle: onToggle,
            ),
          ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add lib/screens/home_tab.dart
git commit -m "feat: add staggered entrance, spring tap, and animated progress to home tab"
```

---

### Task 10: My Day tab — staggered timeline + spring FAB

**Files:**
- Modify: `lib/screens/my_day_tab.dart`

- [ ] **Step 1: Update MyDayTab with staggered items and spring FAB**

Add animation import at top of `lib/screens/my_day_tab.dart`:

```dart
import '../widgets/animations/animations.dart';
```

Then wrap the header and timeline card sections with `StaggeredItem` widgets. Update `_SpeedDialFab` to use `StaggeredItem` for sub-buttons instead of plain `ScaleTransition`.

Replace the `Column` children inside the `SingleChildScrollView` (the content area when `!myDay.isLoading`) — wrap header in `StaggeredItem(index: 0)`, the empty state/timeline card in `StaggeredItem(index: 1)`, completion footer in `StaggeredItem(index: 2)`, and did-you-know in `StaggeredItem(index: 3)`. All with `keepAlive: true`.

In `_SpeedDialFab`, replace the two `ScaleTransition` wrappers with `StaggeredItem(index: 0)` and `StaggeredItem(index: 1)` that are conditionally shown when `_isOpen` is true.

- [ ] **Step 2: Commit**

```bash
git add lib/screens/my_day_tab.dart
git commit -m "feat: add staggered entrances and spring FAB to My Day tab"
```

---

### Task 11: Progress tab — staggered + count-up + animated bars

**Files:**
- Modify: `lib/screens/progress_tab.dart`

- [ ] **Step 1: Update _ProgressContent with StaggeredItem wrappers**

Add animation import to `lib/screens/progress_tab.dart`:

```dart
import '../widgets/animations/animations.dart';
```

Wrap each section (header, key metrics, cause & effect, milestones, medication adherence) in `StaggeredItem` with sequential indices (0-4) and `keepAlive: true`.

- [ ] **Step 2: Commit**

```bash
git add lib/screens/progress_tab.dart
git commit -m "feat: add staggered entrances to Progress tab"
```

---

### Task 12: Learn tab — staggered + spring

**Files:**
- Modify: `lib/screens/learn_tab.dart`

- [ ] **Step 1: Update LearnTab with StaggeredItem wrappers**

Add animation import to `lib/screens/learn_tab.dart`:

```dart
import '../widgets/animations/animations.dart';
```

Wrap header in `StaggeredItem(index: 0)`, alerts in `StaggeredItem(index: 1)`, health tips in `StaggeredItem(index: 2)`, understanding reports in `StaggeredItem(index: 3)`. All with `keepAlive: true`.

- [ ] **Step 2: Commit**

```bash
git add lib/screens/learn_tab.dart
git commit -m "feat: add staggered entrances to Learn tab"
```

---

### Task 13: Settings screen — staggered groups + spring tiles

**Files:**
- Modify: `lib/screens/settings_screen.dart`
- Modify: `lib/widgets/settings_group.dart`
- Modify: `lib/widgets/settings_tile.dart`

- [ ] **Step 1: Wrap SettingsGroup children in StaggeredItem**

In `lib/widgets/settings_group.dart`, add import and wrap in StaggeredItem:

```dart
import '../widgets/animations/animations.dart';
```

The `SettingsGroup` doesn't need changes itself — the stagger will happen at the screen level.

- [ ] **Step 2: Wrap SettingsTile in SpringTapCard**

In `lib/widgets/settings_tile.dart`, add import and wrap the `ListTile` in `SpringTapCard` when `onTap` is not null:

```dart
import 'animations/animations.dart';
```

Replace the build method body: if `onTap != null`, wrap the `ListTile` in a `SpringTapCard(onTap: onTap, ...)`.

- [ ] **Step 3: Update SettingsScreen with StaggeredColumn**

In `lib/screens/settings_screen.dart`, add animation import and wrap the `ListView` children in a `StaggeredColumn` pattern — each `SettingsGroup` gets a `StaggeredItem` with sequential index. Wrap the logout `OutlinedButton` in a `SpringTapCard`.

- [ ] **Step 4: Commit**

```bash
git add lib/screens/settings_screen.dart lib/widgets/settings_tile.dart lib/widgets/settings_group.dart
git commit -m "feat: add staggered groups and spring tap tiles to Settings"
```

---

### Task 14: Score Detail screen — count-up + animated domain bars

**Files:**
- Modify: `lib/screens/score_detail_screen.dart`
- Modify: `lib/widgets/domain_breakdown_bar.dart`
- Modify: `lib/widgets/adherence_ring.dart`

- [ ] **Step 1: Update DomainBreakdownBar to use AnimatedProgressBar**

In `lib/widgets/domain_breakdown_bar.dart`, replace the `TweenAnimationBuilder` + `FractionallySizedBox` score bar with `AnimatedProgressBar`:

Add import:
```dart
import 'animations/animations.dart';
```

Replace the score bar `TweenAnimationBuilder` block (lines 71-85) with:
```dart
AnimatedProgressBar(
  value: score / 100,
  color: color,
  height: 8,
  borderRadius: 4,
),
```

- [ ] **Step 2: Update AdherenceRing to use CountUpText**

In `lib/widgets/adherence_ring.dart`, replace the static `Text('$percentage%')` with `CountUpText(value: percentage.toDouble(), suffix: '%')`.

Add import:
```dart
import 'animations/animations.dart';
```

- [ ] **Step 3: Update ScoreDetailScreen with StaggeredItem wrappers**

In `lib/screens/score_detail_screen.dart`, add animation import and wrap each section (hero ring, sparkline, domain breakdown, explanation) in `StaggeredItem` with sequential indices.

- [ ] **Step 3.5: Add animated line draw to full_sparkline_chart.dart**

In `lib/widgets/full_sparkline_chart.dart`, the sparkline chart should animate its line drawing left-to-right over 800ms using `kDecelerate` curve. Add animation import and wrap the chart painting logic with an `AnimationController` that drives the visible portion of the line from 0% to 100%.

Add import:
```dart
import 'animations/animations.dart';
import '../theme/motion.dart';
```

If the widget is a `StatelessWidget`, convert to `StatefulWidget` with `SingleTickerProviderStateMixin`. Add a controller with 800ms duration, forward on `initState`. Pass `_animation.value` as a clip fraction to the chart painter or use it to limit the `lineBarsData` x-range.

- [ ] **Step 4: Commit**

```bash
git add lib/screens/score_detail_screen.dart lib/widgets/domain_breakdown_bar.dart lib/widgets/adherence_ring.dart lib/widgets/full_sparkline_chart.dart
git commit -m "feat: add count-up text and animated bars to Score Detail and Adherence"
```

---

### Task 15: Notifications screen — staggered + pulse + swipe direction

**Files:**
- Modify: `lib/screens/notifications_screen.dart`
- Modify: `lib/widgets/notification_item.dart`

- [ ] **Step 1: Update NotificationItem swipe direction and pulse**

In `lib/widgets/notification_item.dart`:

Add import:
```dart
import 'animations/animations.dart';
```

1. Change `DismissDirection.endToStart` to `DismissDirection.startToEnd`
2. Move the background to the left side (change `alignment: Alignment.centerRight` to `Alignment.centerLeft`, `padding: EdgeInsets.only(right: 16)` to `EdgeInsets.only(left: 16)`)
3. Wrap the unread dot `Container` in `PulsingWidget`

- [ ] **Step 2: Update NotificationsScreen with staggered groups**

In `lib/screens/notifications_screen.dart`, add animation import. The notification list items are already in groups — wrap the `ListView` items rendering with stagger.

- [ ] **Step 2.5: Add FadeSlideTransition to notification_date_group.dart**

In `lib/widgets/notification_date_group.dart`, add animation import and wrap the date group header `Text` widget in a `FadeSlideTransition` with a slight slide-down effect.

- [ ] **Step 2.6: Add Mark All Read batch animation**

In `lib/screens/notifications_screen.dart`, the "Mark All Read" button (if it exists, or add it to the AppBar actions) should trigger all unread dots to scale down simultaneously. Use a shared `AnimationController` that drives all unread `PulsingWidget` instances to scale to 0 before marking as read.

- [ ] **Step 3: Commit**

```bash
git add lib/screens/notifications_screen.dart lib/widgets/notification_item.dart lib/widgets/notification_date_group.dart
git commit -m "feat: add staggered entrance, pulsing unread dot, and swipe-right dismiss to notifications"
```

---

### Task 16: Vitals entry sheet — glassmorphic + shake

**Files:**
- Modify: `lib/widgets/vitals_entry_sheet.dart`
- Modify: `lib/widgets/bp_entry_card.dart`
- Modify: `lib/widgets/glucose_entry_card.dart`
- Modify: `lib/widgets/weight_entry_card.dart`

- [ ] **Step 1: Update VitalsEntrySheet with GlassmorphicContainer**

In `lib/widgets/vitals_entry_sheet.dart`, add import:
```dart
import 'animations/animations.dart';
```

Replace the `Container` wrapping with `GlassmorphicContainer`. Change `borderRadius` from `Radius.circular(20)` to use default (`24`). Wrap the card list items with `StaggeredItem` indices.

- [ ] **Step 2: Add ShakeWidget to BpEntryCard**

In `lib/widgets/bp_entry_card.dart`, add import and wrap the entire card `Column` in a `ShakeWidget`. Store a `GlobalKey<ShakeWidgetState>` and call `shake()` when validation fails (before or instead of showing error text — keep error text too).

- [ ] **Step 3: Add ShakeWidget to GlucoseEntryCard and WeightEntryCard**

Same pattern as BP: wrap in `ShakeWidget`, call `shake()` on validation failure.

- [ ] **Step 4: Wrap save buttons in SpringTapCard**

In all three entry cards, wrap the `FilledButton` in a `SpringTapCard`.

- [ ] **Step 5: Commit**

```bash
git add lib/widgets/vitals_entry_sheet.dart lib/widgets/bp_entry_card.dart lib/widgets/glucose_entry_card.dart lib/widgets/weight_entry_card.dart
git commit -m "feat: add glassmorphic sheet, shake validation, and spring buttons to vitals entry"
```

---

### Task 17: Symptom logger sheet — glassmorphic + staggered chips

**Files:**
- Modify: `lib/widgets/symptom_logger_sheet.dart`

- [ ] **Step 1: Update SymptomLoggerSheet with GlassmorphicContainer and staggered content**

In `lib/widgets/symptom_logger_sheet.dart`, add import:
```dart
import 'animations/animations.dart';
```

1. Replace the outer `Container` with `GlassmorphicContainer`
2. Wrap the symptom grid, severity selector, notes field, and save button in `StaggeredItem` wrappers with sequential indices
3. Grid stagger pattern: Symptom chips should stagger left-to-right, top-to-bottom. Compute index as `row * cols + col` (assume cols=3 for chip grid). Each chip wrapped in `StaggeredItem(index: row * 3 + col)`.
4. Chip selection: Wrap each chip in `SpringTapCard` for scale bounce on tap. Selected chip border animates with `AnimatedContainer` (200ms duration).
5. Severity selector: Options slide between with `AnimatedSwitcher` + horizontal slide transition.
6. Notes field: Wrap in `FadeSlideTransition` — appears after severity is selected (conditional visibility).
7. Wrap the save `FilledButton` in a `SpringTapCard`

- [ ] **Step 2: Commit**

```bash
git add lib/widgets/symptom_logger_sheet.dart
git commit -m "feat: add glassmorphic sheet and staggered content to symptom logger"
```

---

### Task 18: Medication adherence section — count-up + staggered streaks

**Files:**
- Modify: `lib/widgets/medication_adherence_section.dart`

- [ ] **Step 1: Update MedicationAdherenceSection with staggered streaks**

In `lib/widgets/medication_adherence_section.dart`, add import:
```dart
import 'animations/animations.dart';
```

Wrap the streak rows in a `StaggeredColumn`. The adherence ring already uses `CountUpText` from Task 14.

Missed dose items should slide in from the left (contrasting with right-side streaks) using a `FadeSlideTransition` with negative slide offset (e.g., `slideOffset: -30` for left-to-right entrance). Optionally add a subtle red tint via `Container` with `color: Colors.red.withOpacity(0.03)`.

- [ ] **Step 2: Commit**

```bash
git add lib/widgets/medication_adherence_section.dart
git commit -m "feat: add staggered streaks to medication adherence section"
```

---

### Task 19: Build, test, and verify

**Files:** None (verification only)

- [ ] **Step 1: Run flutter analyze**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter analyze`
Expected: No errors (warnings OK)

- [ ] **Step 2: Run existing tests**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter test`
Expected: All existing tests pass

- [ ] **Step 3: Build for web**

Run: `cd vaidshala/clinical-applications/ui/patient && flutter build web --no-tree-shake-icons`
Expected: Build completes successfully

- [ ] **Step 4: Fix any issues and recommit**

If any analysis errors or test failures, fix them and commit the fixes.

- [ ] **Step 5: Final commit**

```bash
git commit -m "feat: CRED-level motion design retrofit complete — 9 animation widgets, Poppins typography, full app staggered entrances"
```
