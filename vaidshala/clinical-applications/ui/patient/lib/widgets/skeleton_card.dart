import 'package:flutter/material.dart';

class SkeletonCard extends StatefulWidget {
  final double height;

  const SkeletonCard({super.key, required this.height});

  @override
  State<SkeletonCard> createState() => _SkeletonCardState();
}

class _SkeletonCardState extends State<SkeletonCard>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    )..repeat(reverse: true);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, child) {
        final opacity = 0.04 + (_controller.value * 0.06);
        return Card(
          margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
          child: Container(
            height: widget.height,
            decoration: BoxDecoration(
              color: Colors.black.withValues(alpha: opacity),
              borderRadius: BorderRadius.circular(12),
            ),
          ),
        );
      },
    );
  }
}
