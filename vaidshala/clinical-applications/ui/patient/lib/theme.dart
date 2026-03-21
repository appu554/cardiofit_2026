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
