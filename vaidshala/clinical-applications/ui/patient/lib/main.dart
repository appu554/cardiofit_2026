// lib/main.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'router.dart';
import 'services/hive_service.dart';
import 'theme.dart';
import 'providers/locale_provider.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await HiveService.init();
  runApp(const ProviderScope(child: VaidshalaPatientApp()));
}

class VaidshalaPatientApp extends ConsumerWidget {
  const VaidshalaPatientApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    final locale = ref.watch(localeProvider);

    return MaterialApp.router(
      title: 'Vaidshala',
      debugShowCheckedModeBanner: false,
      theme: buildAppTheme(),
      locale: locale,
      routerConfig: router,
    );
  }
}
