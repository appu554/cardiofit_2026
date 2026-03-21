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
