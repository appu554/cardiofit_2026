import 'package:freezed_annotation/freezed_annotation.dart';

part 'driver.freezed.dart';
part 'driver.g.dart';

@freezed
class HealthDriver with _$HealthDriver {
  const factory HealthDriver({
    required String id,
    required String name,
    required String icon,
    required double current,
    required double target,
    required String unit,
    required bool improving,
  }) = _HealthDriver;

  factory HealthDriver.fromJson(Map<String, dynamic> json) =>
      _$HealthDriverFromJson(json);
}
