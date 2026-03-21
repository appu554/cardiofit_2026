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
