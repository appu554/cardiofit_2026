// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'cause_effect.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

CauseEffect _$CauseEffectFromJson(Map<String, dynamic> json) {
  return _CauseEffect.fromJson(json);
}

/// @nodoc
mixin _$CauseEffect {
  String get id => throw _privateConstructorUsedError;
  String get cause => throw _privateConstructorUsedError;
  String get effect => throw _privateConstructorUsedError;
  String get causeIcon => throw _privateConstructorUsedError;
  String get effectIcon => throw _privateConstructorUsedError;
  bool get verified => throw _privateConstructorUsedError;

  /// Serializes this CauseEffect to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CauseEffect
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CauseEffectCopyWith<CauseEffect> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CauseEffectCopyWith<$Res> {
  factory $CauseEffectCopyWith(
    CauseEffect value,
    $Res Function(CauseEffect) then,
  ) = _$CauseEffectCopyWithImpl<$Res, CauseEffect>;
  @useResult
  $Res call({
    String id,
    String cause,
    String effect,
    String causeIcon,
    String effectIcon,
    bool verified,
  });
}

/// @nodoc
class _$CauseEffectCopyWithImpl<$Res, $Val extends CauseEffect>
    implements $CauseEffectCopyWith<$Res> {
  _$CauseEffectCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CauseEffect
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? cause = null,
    Object? effect = null,
    Object? causeIcon = null,
    Object? effectIcon = null,
    Object? verified = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as String,
            cause: null == cause
                ? _value.cause
                : cause // ignore: cast_nullable_to_non_nullable
                      as String,
            effect: null == effect
                ? _value.effect
                : effect // ignore: cast_nullable_to_non_nullable
                      as String,
            causeIcon: null == causeIcon
                ? _value.causeIcon
                : causeIcon // ignore: cast_nullable_to_non_nullable
                      as String,
            effectIcon: null == effectIcon
                ? _value.effectIcon
                : effectIcon // ignore: cast_nullable_to_non_nullable
                      as String,
            verified: null == verified
                ? _value.verified
                : verified // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CauseEffectImplCopyWith<$Res>
    implements $CauseEffectCopyWith<$Res> {
  factory _$$CauseEffectImplCopyWith(
    _$CauseEffectImpl value,
    $Res Function(_$CauseEffectImpl) then,
  ) = __$$CauseEffectImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String id,
    String cause,
    String effect,
    String causeIcon,
    String effectIcon,
    bool verified,
  });
}

/// @nodoc
class __$$CauseEffectImplCopyWithImpl<$Res>
    extends _$CauseEffectCopyWithImpl<$Res, _$CauseEffectImpl>
    implements _$$CauseEffectImplCopyWith<$Res> {
  __$$CauseEffectImplCopyWithImpl(
    _$CauseEffectImpl _value,
    $Res Function(_$CauseEffectImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CauseEffect
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? cause = null,
    Object? effect = null,
    Object? causeIcon = null,
    Object? effectIcon = null,
    Object? verified = null,
  }) {
    return _then(
      _$CauseEffectImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as String,
        cause: null == cause
            ? _value.cause
            : cause // ignore: cast_nullable_to_non_nullable
                  as String,
        effect: null == effect
            ? _value.effect
            : effect // ignore: cast_nullable_to_non_nullable
                  as String,
        causeIcon: null == causeIcon
            ? _value.causeIcon
            : causeIcon // ignore: cast_nullable_to_non_nullable
                  as String,
        effectIcon: null == effectIcon
            ? _value.effectIcon
            : effectIcon // ignore: cast_nullable_to_non_nullable
                  as String,
        verified: null == verified
            ? _value.verified
            : verified // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CauseEffectImpl implements _CauseEffect {
  const _$CauseEffectImpl({
    required this.id,
    required this.cause,
    required this.effect,
    required this.causeIcon,
    required this.effectIcon,
    this.verified = false,
  });

  factory _$CauseEffectImpl.fromJson(Map<String, dynamic> json) =>
      _$$CauseEffectImplFromJson(json);

  @override
  final String id;
  @override
  final String cause;
  @override
  final String effect;
  @override
  final String causeIcon;
  @override
  final String effectIcon;
  @override
  @JsonKey()
  final bool verified;

  @override
  String toString() {
    return 'CauseEffect(id: $id, cause: $cause, effect: $effect, causeIcon: $causeIcon, effectIcon: $effectIcon, verified: $verified)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CauseEffectImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.cause, cause) || other.cause == cause) &&
            (identical(other.effect, effect) || other.effect == effect) &&
            (identical(other.causeIcon, causeIcon) ||
                other.causeIcon == causeIcon) &&
            (identical(other.effectIcon, effectIcon) ||
                other.effectIcon == effectIcon) &&
            (identical(other.verified, verified) ||
                other.verified == verified));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    cause,
    effect,
    causeIcon,
    effectIcon,
    verified,
  );

  /// Create a copy of CauseEffect
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CauseEffectImplCopyWith<_$CauseEffectImpl> get copyWith =>
      __$$CauseEffectImplCopyWithImpl<_$CauseEffectImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CauseEffectImplToJson(this);
  }
}

abstract class _CauseEffect implements CauseEffect {
  const factory _CauseEffect({
    required final String id,
    required final String cause,
    required final String effect,
    required final String causeIcon,
    required final String effectIcon,
    final bool verified,
  }) = _$CauseEffectImpl;

  factory _CauseEffect.fromJson(Map<String, dynamic> json) =
      _$CauseEffectImpl.fromJson;

  @override
  String get id;
  @override
  String get cause;
  @override
  String get effect;
  @override
  String get causeIcon;
  @override
  String get effectIcon;
  @override
  bool get verified;

  /// Create a copy of CauseEffect
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CauseEffectImplCopyWith<_$CauseEffectImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
