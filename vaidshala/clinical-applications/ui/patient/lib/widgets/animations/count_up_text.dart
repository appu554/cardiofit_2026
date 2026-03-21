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
