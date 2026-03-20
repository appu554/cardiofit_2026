// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'insight.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

Insight _$InsightFromJson(Map<String, dynamic> json) {
  return _Insight.fromJson(json);
}

/// @nodoc
mixin _$Insight {
  String? get coachingMessage => throw _privateConstructorUsedError;
  InsightType? get coachingType => throw _privateConstructorUsedError;
  List<String> get tips => throw _privateConstructorUsedError;
  List<String> get alerts => throw _privateConstructorUsedError;

  /// Serializes this Insight to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of Insight
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $InsightCopyWith<Insight> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $InsightCopyWith<$Res> {
  factory $InsightCopyWith(Insight value, $Res Function(Insight) then) =
      _$InsightCopyWithImpl<$Res, Insight>;
  @useResult
  $Res call({
    String? coachingMessage,
    InsightType? coachingType,
    List<String> tips,
    List<String> alerts,
  });
}

/// @nodoc
class _$InsightCopyWithImpl<$Res, $Val extends Insight>
    implements $InsightCopyWith<$Res> {
  _$InsightCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of Insight
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? coachingMessage = freezed,
    Object? coachingType = freezed,
    Object? tips = null,
    Object? alerts = null,
  }) {
    return _then(
      _value.copyWith(
            coachingMessage: freezed == coachingMessage
                ? _value.coachingMessage
                : coachingMessage // ignore: cast_nullable_to_non_nullable
                      as String?,
            coachingType: freezed == coachingType
                ? _value.coachingType
                : coachingType // ignore: cast_nullable_to_non_nullable
                      as InsightType?,
            tips: null == tips
                ? _value.tips
                : tips // ignore: cast_nullable_to_non_nullable
                      as List<String>,
            alerts: null == alerts
                ? _value.alerts
                : alerts // ignore: cast_nullable_to_non_nullable
                      as List<String>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$InsightImplCopyWith<$Res> implements $InsightCopyWith<$Res> {
  factory _$$InsightImplCopyWith(
    _$InsightImpl value,
    $Res Function(_$InsightImpl) then,
  ) = __$$InsightImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String? coachingMessage,
    InsightType? coachingType,
    List<String> tips,
    List<String> alerts,
  });
}

/// @nodoc
class __$$InsightImplCopyWithImpl<$Res>
    extends _$InsightCopyWithImpl<$Res, _$InsightImpl>
    implements _$$InsightImplCopyWith<$Res> {
  __$$InsightImplCopyWithImpl(
    _$InsightImpl _value,
    $Res Function(_$InsightImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of Insight
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? coachingMessage = freezed,
    Object? coachingType = freezed,
    Object? tips = null,
    Object? alerts = null,
  }) {
    return _then(
      _$InsightImpl(
        coachingMessage: freezed == coachingMessage
            ? _value.coachingMessage
            : coachingMessage // ignore: cast_nullable_to_non_nullable
                  as String?,
        coachingType: freezed == coachingType
            ? _value.coachingType
            : coachingType // ignore: cast_nullable_to_non_nullable
                  as InsightType?,
        tips: null == tips
            ? _value._tips
            : tips // ignore: cast_nullable_to_non_nullable
                  as List<String>,
        alerts: null == alerts
            ? _value._alerts
            : alerts // ignore: cast_nullable_to_non_nullable
                  as List<String>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$InsightImpl implements _Insight {
  const _$InsightImpl({
    this.coachingMessage,
    this.coachingType,
    final List<String> tips = const [],
    final List<String> alerts = const [],
  }) : _tips = tips,
       _alerts = alerts;

  factory _$InsightImpl.fromJson(Map<String, dynamic> json) =>
      _$$InsightImplFromJson(json);

  @override
  final String? coachingMessage;
  @override
  final InsightType? coachingType;
  final List<String> _tips;
  @override
  @JsonKey()
  List<String> get tips {
    if (_tips is EqualUnmodifiableListView) return _tips;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_tips);
  }

  final List<String> _alerts;
  @override
  @JsonKey()
  List<String> get alerts {
    if (_alerts is EqualUnmodifiableListView) return _alerts;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_alerts);
  }

  @override
  String toString() {
    return 'Insight(coachingMessage: $coachingMessage, coachingType: $coachingType, tips: $tips, alerts: $alerts)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$InsightImpl &&
            (identical(other.coachingMessage, coachingMessage) ||
                other.coachingMessage == coachingMessage) &&
            (identical(other.coachingType, coachingType) ||
                other.coachingType == coachingType) &&
            const DeepCollectionEquality().equals(other._tips, _tips) &&
            const DeepCollectionEquality().equals(other._alerts, _alerts));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    coachingMessage,
    coachingType,
    const DeepCollectionEquality().hash(_tips),
    const DeepCollectionEquality().hash(_alerts),
  );

  /// Create a copy of Insight
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$InsightImplCopyWith<_$InsightImpl> get copyWith =>
      __$$InsightImplCopyWithImpl<_$InsightImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$InsightImplToJson(this);
  }
}

abstract class _Insight implements Insight {
  const factory _Insight({
    final String? coachingMessage,
    final InsightType? coachingType,
    final List<String> tips,
    final List<String> alerts,
  }) = _$InsightImpl;

  factory _Insight.fromJson(Map<String, dynamic> json) = _$InsightImpl.fromJson;

  @override
  String? get coachingMessage;
  @override
  InsightType? get coachingType;
  @override
  List<String> get tips;
  @override
  List<String> get alerts;

  /// Create a copy of Insight
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$InsightImplCopyWith<_$InsightImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
