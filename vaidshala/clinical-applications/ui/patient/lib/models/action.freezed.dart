// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'action.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

PatientAction _$PatientActionFromJson(Map<String, dynamic> json) {
  return _PatientAction.fromJson(json);
}

/// @nodoc
mixin _$PatientAction {
  String get id => throw _privateConstructorUsedError;
  String get text => throw _privateConstructorUsedError;
  String get icon => throw _privateConstructorUsedError;
  String get time => throw _privateConstructorUsedError;
  bool get done => throw _privateConstructorUsedError;
  String? get why => throw _privateConstructorUsedError;

  /// Serializes this PatientAction to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PatientAction
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PatientActionCopyWith<PatientAction> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PatientActionCopyWith<$Res> {
  factory $PatientActionCopyWith(
    PatientAction value,
    $Res Function(PatientAction) then,
  ) = _$PatientActionCopyWithImpl<$Res, PatientAction>;
  @useResult
  $Res call({
    String id,
    String text,
    String icon,
    String time,
    bool done,
    String? why,
  });
}

/// @nodoc
class _$PatientActionCopyWithImpl<$Res, $Val extends PatientAction>
    implements $PatientActionCopyWith<$Res> {
  _$PatientActionCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PatientAction
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? text = null,
    Object? icon = null,
    Object? time = null,
    Object? done = null,
    Object? why = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as String,
            text: null == text
                ? _value.text
                : text // ignore: cast_nullable_to_non_nullable
                      as String,
            icon: null == icon
                ? _value.icon
                : icon // ignore: cast_nullable_to_non_nullable
                      as String,
            time: null == time
                ? _value.time
                : time // ignore: cast_nullable_to_non_nullable
                      as String,
            done: null == done
                ? _value.done
                : done // ignore: cast_nullable_to_non_nullable
                      as bool,
            why: freezed == why
                ? _value.why
                : why // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$PatientActionImplCopyWith<$Res>
    implements $PatientActionCopyWith<$Res> {
  factory _$$PatientActionImplCopyWith(
    _$PatientActionImpl value,
    $Res Function(_$PatientActionImpl) then,
  ) = __$$PatientActionImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String id,
    String text,
    String icon,
    String time,
    bool done,
    String? why,
  });
}

/// @nodoc
class __$$PatientActionImplCopyWithImpl<$Res>
    extends _$PatientActionCopyWithImpl<$Res, _$PatientActionImpl>
    implements _$$PatientActionImplCopyWith<$Res> {
  __$$PatientActionImplCopyWithImpl(
    _$PatientActionImpl _value,
    $Res Function(_$PatientActionImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PatientAction
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? text = null,
    Object? icon = null,
    Object? time = null,
    Object? done = null,
    Object? why = freezed,
  }) {
    return _then(
      _$PatientActionImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as String,
        text: null == text
            ? _value.text
            : text // ignore: cast_nullable_to_non_nullable
                  as String,
        icon: null == icon
            ? _value.icon
            : icon // ignore: cast_nullable_to_non_nullable
                  as String,
        time: null == time
            ? _value.time
            : time // ignore: cast_nullable_to_non_nullable
                  as String,
        done: null == done
            ? _value.done
            : done // ignore: cast_nullable_to_non_nullable
                  as bool,
        why: freezed == why
            ? _value.why
            : why // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$PatientActionImpl implements _PatientAction {
  const _$PatientActionImpl({
    required this.id,
    required this.text,
    required this.icon,
    required this.time,
    this.done = false,
    this.why,
  });

  factory _$PatientActionImpl.fromJson(Map<String, dynamic> json) =>
      _$$PatientActionImplFromJson(json);

  @override
  final String id;
  @override
  final String text;
  @override
  final String icon;
  @override
  final String time;
  @override
  @JsonKey()
  final bool done;
  @override
  final String? why;

  @override
  String toString() {
    return 'PatientAction(id: $id, text: $text, icon: $icon, time: $time, done: $done, why: $why)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PatientActionImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.text, text) || other.text == text) &&
            (identical(other.icon, icon) || other.icon == icon) &&
            (identical(other.time, time) || other.time == time) &&
            (identical(other.done, done) || other.done == done) &&
            (identical(other.why, why) || other.why == why));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, text, icon, time, done, why);

  /// Create a copy of PatientAction
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PatientActionImplCopyWith<_$PatientActionImpl> get copyWith =>
      __$$PatientActionImplCopyWithImpl<_$PatientActionImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$PatientActionImplToJson(this);
  }
}

abstract class _PatientAction implements PatientAction {
  const factory _PatientAction({
    required final String id,
    required final String text,
    required final String icon,
    required final String time,
    final bool done,
    final String? why,
  }) = _$PatientActionImpl;

  factory _PatientAction.fromJson(Map<String, dynamic> json) =
      _$PatientActionImpl.fromJson;

  @override
  String get id;
  @override
  String get text;
  @override
  String get icon;
  @override
  String get time;
  @override
  bool get done;
  @override
  String? get why;

  /// Create a copy of PatientAction
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PatientActionImplCopyWith<_$PatientActionImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
