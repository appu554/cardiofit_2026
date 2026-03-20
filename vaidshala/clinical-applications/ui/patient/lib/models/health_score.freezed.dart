// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'health_score.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

HealthScore _$HealthScoreFromJson(Map<String, dynamic> json) {
  return _HealthScore.fromJson(json);
}

/// @nodoc
mixin _$HealthScore {
  int get score => throw _privateConstructorUsedError;
  String get label => throw _privateConstructorUsedError;
  int get delta => throw _privateConstructorUsedError;
  List<int> get sparkline => throw _privateConstructorUsedError;
  DateTime? get updatedAt => throw _privateConstructorUsedError;

  /// Serializes this HealthScore to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of HealthScore
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $HealthScoreCopyWith<HealthScore> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $HealthScoreCopyWith<$Res> {
  factory $HealthScoreCopyWith(
    HealthScore value,
    $Res Function(HealthScore) then,
  ) = _$HealthScoreCopyWithImpl<$Res, HealthScore>;
  @useResult
  $Res call({
    int score,
    String label,
    int delta,
    List<int> sparkline,
    DateTime? updatedAt,
  });
}

/// @nodoc
class _$HealthScoreCopyWithImpl<$Res, $Val extends HealthScore>
    implements $HealthScoreCopyWith<$Res> {
  _$HealthScoreCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of HealthScore
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? score = null,
    Object? label = null,
    Object? delta = null,
    Object? sparkline = null,
    Object? updatedAt = freezed,
  }) {
    return _then(
      _value.copyWith(
            score: null == score
                ? _value.score
                : score // ignore: cast_nullable_to_non_nullable
                      as int,
            label: null == label
                ? _value.label
                : label // ignore: cast_nullable_to_non_nullable
                      as String,
            delta: null == delta
                ? _value.delta
                : delta // ignore: cast_nullable_to_non_nullable
                      as int,
            sparkline: null == sparkline
                ? _value.sparkline
                : sparkline // ignore: cast_nullable_to_non_nullable
                      as List<int>,
            updatedAt: freezed == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as DateTime?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$HealthScoreImplCopyWith<$Res>
    implements $HealthScoreCopyWith<$Res> {
  factory _$$HealthScoreImplCopyWith(
    _$HealthScoreImpl value,
    $Res Function(_$HealthScoreImpl) then,
  ) = __$$HealthScoreImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int score,
    String label,
    int delta,
    List<int> sparkline,
    DateTime? updatedAt,
  });
}

/// @nodoc
class __$$HealthScoreImplCopyWithImpl<$Res>
    extends _$HealthScoreCopyWithImpl<$Res, _$HealthScoreImpl>
    implements _$$HealthScoreImplCopyWith<$Res> {
  __$$HealthScoreImplCopyWithImpl(
    _$HealthScoreImpl _value,
    $Res Function(_$HealthScoreImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of HealthScore
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? score = null,
    Object? label = null,
    Object? delta = null,
    Object? sparkline = null,
    Object? updatedAt = freezed,
  }) {
    return _then(
      _$HealthScoreImpl(
        score: null == score
            ? _value.score
            : score // ignore: cast_nullable_to_non_nullable
                  as int,
        label: null == label
            ? _value.label
            : label // ignore: cast_nullable_to_non_nullable
                  as String,
        delta: null == delta
            ? _value.delta
            : delta // ignore: cast_nullable_to_non_nullable
                  as int,
        sparkline: null == sparkline
            ? _value._sparkline
            : sparkline // ignore: cast_nullable_to_non_nullable
                  as List<int>,
        updatedAt: freezed == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as DateTime?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$HealthScoreImpl implements _HealthScore {
  const _$HealthScoreImpl({
    required this.score,
    required this.label,
    this.delta = 0,
    final List<int> sparkline = const [],
    this.updatedAt,
  }) : _sparkline = sparkline;

  factory _$HealthScoreImpl.fromJson(Map<String, dynamic> json) =>
      _$$HealthScoreImplFromJson(json);

  @override
  final int score;
  @override
  final String label;
  @override
  @JsonKey()
  final int delta;
  final List<int> _sparkline;
  @override
  @JsonKey()
  List<int> get sparkline {
    if (_sparkline is EqualUnmodifiableListView) return _sparkline;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_sparkline);
  }

  @override
  final DateTime? updatedAt;

  @override
  String toString() {
    return 'HealthScore(score: $score, label: $label, delta: $delta, sparkline: $sparkline, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$HealthScoreImpl &&
            (identical(other.score, score) || other.score == score) &&
            (identical(other.label, label) || other.label == label) &&
            (identical(other.delta, delta) || other.delta == delta) &&
            const DeepCollectionEquality().equals(
              other._sparkline,
              _sparkline,
            ) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    score,
    label,
    delta,
    const DeepCollectionEquality().hash(_sparkline),
    updatedAt,
  );

  /// Create a copy of HealthScore
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$HealthScoreImplCopyWith<_$HealthScoreImpl> get copyWith =>
      __$$HealthScoreImplCopyWithImpl<_$HealthScoreImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$HealthScoreImplToJson(this);
  }
}

abstract class _HealthScore implements HealthScore {
  const factory _HealthScore({
    required final int score,
    required final String label,
    final int delta,
    final List<int> sparkline,
    final DateTime? updatedAt,
  }) = _$HealthScoreImpl;

  factory _HealthScore.fromJson(Map<String, dynamic> json) =
      _$HealthScoreImpl.fromJson;

  @override
  int get score;
  @override
  String get label;
  @override
  int get delta;
  @override
  List<int> get sparkline;
  @override
  DateTime? get updatedAt;

  /// Create a copy of HealthScore
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$HealthScoreImplCopyWith<_$HealthScoreImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
