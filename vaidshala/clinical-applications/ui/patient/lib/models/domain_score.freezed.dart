// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'domain_score.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

DomainScore _$DomainScoreFromJson(Map<String, dynamic> json) {
  return _DomainScore.fromJson(json);
}

/// @nodoc
mixin _$DomainScore {
  String get name => throw _privateConstructorUsedError;
  int get score => throw _privateConstructorUsedError;
  int get target => throw _privateConstructorUsedError;
  String get icon => throw _privateConstructorUsedError;

  /// Serializes this DomainScore to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of DomainScore
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $DomainScoreCopyWith<DomainScore> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $DomainScoreCopyWith<$Res> {
  factory $DomainScoreCopyWith(
    DomainScore value,
    $Res Function(DomainScore) then,
  ) = _$DomainScoreCopyWithImpl<$Res, DomainScore>;
  @useResult
  $Res call({String name, int score, int target, String icon});
}

/// @nodoc
class _$DomainScoreCopyWithImpl<$Res, $Val extends DomainScore>
    implements $DomainScoreCopyWith<$Res> {
  _$DomainScoreCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of DomainScore
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? score = null,
    Object? target = null,
    Object? icon = null,
  }) {
    return _then(
      _value.copyWith(
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            score: null == score
                ? _value.score
                : score // ignore: cast_nullable_to_non_nullable
                      as int,
            target: null == target
                ? _value.target
                : target // ignore: cast_nullable_to_non_nullable
                      as int,
            icon: null == icon
                ? _value.icon
                : icon // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$DomainScoreImplCopyWith<$Res>
    implements $DomainScoreCopyWith<$Res> {
  factory _$$DomainScoreImplCopyWith(
    _$DomainScoreImpl value,
    $Res Function(_$DomainScoreImpl) then,
  ) = __$$DomainScoreImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String name, int score, int target, String icon});
}

/// @nodoc
class __$$DomainScoreImplCopyWithImpl<$Res>
    extends _$DomainScoreCopyWithImpl<$Res, _$DomainScoreImpl>
    implements _$$DomainScoreImplCopyWith<$Res> {
  __$$DomainScoreImplCopyWithImpl(
    _$DomainScoreImpl _value,
    $Res Function(_$DomainScoreImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of DomainScore
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? score = null,
    Object? target = null,
    Object? icon = null,
  }) {
    return _then(
      _$DomainScoreImpl(
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        score: null == score
            ? _value.score
            : score // ignore: cast_nullable_to_non_nullable
                  as int,
        target: null == target
            ? _value.target
            : target // ignore: cast_nullable_to_non_nullable
                  as int,
        icon: null == icon
            ? _value.icon
            : icon // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$DomainScoreImpl implements _DomainScore {
  const _$DomainScoreImpl({
    required this.name,
    required this.score,
    required this.target,
    required this.icon,
  });

  factory _$DomainScoreImpl.fromJson(Map<String, dynamic> json) =>
      _$$DomainScoreImplFromJson(json);

  @override
  final String name;
  @override
  final int score;
  @override
  final int target;
  @override
  final String icon;

  @override
  String toString() {
    return 'DomainScore(name: $name, score: $score, target: $target, icon: $icon)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$DomainScoreImpl &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.score, score) || other.score == score) &&
            (identical(other.target, target) || other.target == target) &&
            (identical(other.icon, icon) || other.icon == icon));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, name, score, target, icon);

  /// Create a copy of DomainScore
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$DomainScoreImplCopyWith<_$DomainScoreImpl> get copyWith =>
      __$$DomainScoreImplCopyWithImpl<_$DomainScoreImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$DomainScoreImplToJson(this);
  }
}

abstract class _DomainScore implements DomainScore {
  const factory _DomainScore({
    required final String name,
    required final int score,
    required final int target,
    required final String icon,
  }) = _$DomainScoreImpl;

  factory _DomainScore.fromJson(Map<String, dynamic> json) =
      _$DomainScoreImpl.fromJson;

  @override
  String get name;
  @override
  int get score;
  @override
  int get target;
  @override
  String get icon;

  /// Create a copy of DomainScore
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$DomainScoreImplCopyWith<_$DomainScoreImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
