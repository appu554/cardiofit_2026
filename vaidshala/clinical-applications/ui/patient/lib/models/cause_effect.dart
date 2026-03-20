import 'package:freezed_annotation/freezed_annotation.dart';

part 'cause_effect.freezed.dart';
part 'cause_effect.g.dart';

@freezed
class CauseEffect with _$CauseEffect {
  const factory CauseEffect({
    required String id,
    required String cause,
    required String effect,
    required String causeIcon,
    required String effectIcon,
    @Default(false) bool verified,
  }) = _CauseEffect;

  factory CauseEffect.fromJson(Map<String, dynamic> json) =>
      _$CauseEffectFromJson(json);
}
