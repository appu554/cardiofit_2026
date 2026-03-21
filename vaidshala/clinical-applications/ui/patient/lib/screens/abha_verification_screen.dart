import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/abha_provider.dart';
import '../theme.dart';

class AbhaVerificationScreen extends ConsumerWidget {
  const AbhaVerificationScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final abha = ref.watch(abhaProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('ABHA Verification'),
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: () => context.go('/home/dashboard'),
        ),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Explanation
            const Text(
              'Link Your ABHA ID',
              style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            const Text(
              'ABHA (Ayushman Bharat Health Account) allows secure sharing of your health records across healthcare providers.',
              style: TextStyle(
                  fontSize: 14, color: AppColors.textSecondary, height: 1.5),
            ),
            const SizedBox(height: 24),

            // Success state
            if (abha.status == AbhaStatus.success) ...[
              Card(
                color: AppColors.coachingGreen,
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    children: [
                      const Icon(Icons.check_circle,
                          color: AppColors.scoreGreen, size: 48),
                      const SizedBox(height: 12),
                      const Text(
                        'ABHA Verified!',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                          color: AppColors.scoreGreen,
                        ),
                      ),
                      if (abha.phrAddress != null) ...[
                        const SizedBox(height: 4),
                        Text(
                          'PHR: ${abha.phrAddress}',
                          style: const TextStyle(
                              fontSize: 14, color: AppColors.textSecondary),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 24),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () => context.go('/home/dashboard'),
                  child: const Text('Continue to Dashboard'),
                ),
              ),
            ] else ...[
              // ABHA Input
              TextFormField(
                decoration: const InputDecoration(
                  labelText: 'ABHA ID',
                  hintText: 'XX-XXXX-XXXX-XXXX',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.badge),
                ),
                keyboardType: TextInputType.number,
                inputFormatters: [
                  FilteringTextInputFormatter.allow(RegExp(r'[\d-]')),
                  LengthLimitingTextInputFormatter(17),
                ],
                onChanged: (v) =>
                    ref.read(abhaProvider.notifier).setAbhaId(v),
              ),

              if (abha.error != null) ...[
                const SizedBox(height: 8),
                Text(
                  abha.error!,
                  style: const TextStyle(
                      color: AppColors.scoreRed, fontSize: 13),
                ),
              ],

              const SizedBox(height: 20),

              // Verify button
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: abha.status == AbhaStatus.verifying
                      ? null
                      : () => ref.read(abhaProvider.notifier).verify(),
                  child: abha.status == AbhaStatus.verifying
                      ? const SizedBox(
                          height: 20,
                          width: 20,
                          child: CircularProgressIndicator(
                              strokeWidth: 2, color: Colors.white),
                        )
                      : const Text('Verify ABHA'),
                ),
              ),

              const SizedBox(height: 12),

              // Skip button
              SizedBox(
                width: double.infinity,
                child: TextButton(
                  onPressed: () => context.go('/home/dashboard'),
                  child: const Text('Skip for now'),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
