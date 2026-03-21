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
