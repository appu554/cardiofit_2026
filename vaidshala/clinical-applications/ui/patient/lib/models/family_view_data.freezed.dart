// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'family_view_data.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

FamilyViewData _$FamilyViewDataFromJson(Map<String, dynamic> json) {
  return _FamilyViewData.fromJson(json);
}

/// @nodoc
mixin _$FamilyViewData {
  String get patientName => throw _privateConstructorUsedError;
  List<FamilyAction> get mealActions => throw _privateConstructorUsedError;
  List<FamilyAction> get activityActions => throw _privateConstructorUsedError;
  String? get supportMessage => throw _privateConstructorUsedError;

  /// Serializes this FamilyViewData to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of FamilyViewData
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $FamilyViewDataCopyWith<FamilyViewData> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $FamilyViewDataCopyWith<$Res> {
  factory $FamilyViewDataCopyWith(
    FamilyViewData value,
    $Res Function(FamilyViewData) then,
  ) = _$FamilyViewDataCopyWithImpl<$Res, FamilyViewData>;
  @useResult
  $Res call({
    String patientName,
    List<FamilyAction> mealActions,
    List<FamilyAction> activityActions,
    String? supportMessage,
  });
}

/// @nodoc
class _$FamilyViewDataCopyWithImpl<$Res, $Val extends FamilyViewData>
    implements $FamilyViewDataCopyWith<$Res> {
  _$FamilyViewDataCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of FamilyViewData
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? patientName = null,
    Object? mealActions = null,
    Object? activityActions = null,
    Object? supportMessage = freezed,
  }) {
    return _then(
      _value.copyWith(
            patientName: null == patientName
                ? _value.patientName
                : patientName // ignore: cast_nullable_to_non_nullable
                      as String,
            mealActions: null == mealActions
                ? _value.mealActions
                : mealActions // ignore: cast_nullable_to_non_nullable
                      as List<FamilyAction>,
            activityActions: null == activityActions
                ? _value.activityActions
                : activityActions // ignore: cast_nullable_to_non_nullable
                      as List<FamilyAction>,
            supportMessage: freezed == supportMessage
                ? _value.supportMessage
                : supportMessage // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$FamilyViewDataImplCopyWith<$Res>
    implements $FamilyViewDataCopyWith<$Res> {
  factory _$$FamilyViewDataImplCopyWith(
    _$FamilyViewDataImpl value,
    $Res Function(_$FamilyViewDataImpl) then,
  ) = __$$FamilyViewDataImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String patientName,
    List<FamilyAction> mealActions,
    List<FamilyAction> activityActions,
    String? supportMessage,
  });
}

/// @nodoc
class __$$FamilyViewDataImplCopyWithImpl<$Res>
    extends _$FamilyViewDataCopyWithImpl<$Res, _$FamilyViewDataImpl>
    implements _$$FamilyViewDataImplCopyWith<$Res> {
  __$$FamilyViewDataImplCopyWithImpl(
    _$FamilyViewDataImpl _value,
    $Res Function(_$FamilyViewDataImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of FamilyViewData
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? patientName = null,
    Object? mealActions = null,
    Object? activityActions = null,
    Object? supportMessage = freezed,
  }) {
    return _then(
      _$FamilyViewDataImpl(
        patientName: null == patientName
            ? _value.patientName
            : patientName // ignore: cast_nullable_to_non_nullable
                  as String,
        mealActions: null == mealActions
            ? _value._mealActions
            : mealActions // ignore: cast_nullable_to_non_nullable
                  as List<FamilyAction>,
        activityActions: null == activityActions
            ? _value._activityActions
            : activityActions // ignore: cast_nullable_to_non_nullable
                  as List<FamilyAction>,
        supportMessage: freezed == supportMessage
            ? _value.supportMessage
            : supportMessage // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$FamilyViewDataImpl implements _FamilyViewData {
  const _$FamilyViewDataImpl({
    required this.patientName,
    final List<FamilyAction> mealActions = const [],
    final List<FamilyAction> activityActions = const [],
    this.supportMessage,
  }) : _mealActions = mealActions,
       _activityActions = activityActions;

  factory _$FamilyViewDataImpl.fromJson(Map<String, dynamic> json) =>
      _$$FamilyViewDataImplFromJson(json);

  @override
  final String patientName;
  final List<FamilyAction> _mealActions;
  @override
  @JsonKey()
  List<FamilyAction> get mealActions {
    if (_mealActions is EqualUnmodifiableListView) return _mealActions;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_mealActions);
  }

  final List<FamilyAction> _activityActions;
  @override
  @JsonKey()
  List<FamilyAction> get activityActions {
    if (_activityActions is EqualUnmodifiableListView) return _activityActions;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_activityActions);
  }

  @override
  final String? supportMessage;

  @override
  String toString() {
    return 'FamilyViewData(patientName: $patientName, mealActions: $mealActions, activityActions: $activityActions, supportMessage: $supportMessage)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$FamilyViewDataImpl &&
            (identical(other.patientName, patientName) ||
                other.patientName == patientName) &&
            const DeepCollectionEquality().equals(
              other._mealActions,
              _mealActions,
            ) &&
            const DeepCollectionEquality().equals(
              other._activityActions,
              _activityActions,
            ) &&
            (identical(other.supportMessage, supportMessage) ||
                other.supportMessage == supportMessage));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    patientName,
    const DeepCollectionEquality().hash(_mealActions),
    const DeepCollectionEquality().hash(_activityActions),
    supportMessage,
  );

  /// Create a copy of FamilyViewData
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$FamilyViewDataImplCopyWith<_$FamilyViewDataImpl> get copyWith =>
      __$$FamilyViewDataImplCopyWithImpl<_$FamilyViewDataImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$FamilyViewDataImplToJson(this);
  }
}

abstract class _FamilyViewData implements FamilyViewData {
  const factory _FamilyViewData({
    required final String patientName,
    final List<FamilyAction> mealActions,
    final List<FamilyAction> activityActions,
    final String? supportMessage,
  }) = _$FamilyViewDataImpl;

  factory _FamilyViewData.fromJson(Map<String, dynamic> json) =
      _$FamilyViewDataImpl.fromJson;

  @override
  String get patientName;
  @override
  List<FamilyAction> get mealActions;
  @override
  List<FamilyAction> get activityActions;
  @override
  String? get supportMessage;

  /// Create a copy of FamilyViewData
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$FamilyViewDataImplCopyWith<_$FamilyViewDataImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

FamilyAction _$FamilyActionFromJson(Map<String, dynamic> json) {
  return _FamilyAction.fromJson(json);
}

/// @nodoc
mixin _$FamilyAction {
  String get text => throw _privateConstructorUsedError;
  String get icon => throw _privateConstructorUsedError;
  String get time => throw _privateConstructorUsedError;

  /// Serializes this FamilyAction to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of FamilyAction
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $FamilyActionCopyWith<FamilyAction> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $FamilyActionCopyWith<$Res> {
  factory $FamilyActionCopyWith(
    FamilyAction value,
    $Res Function(FamilyAction) then,
  ) = _$FamilyActionCopyWithImpl<$Res, FamilyAction>;
  @useResult
  $Res call({String text, String icon, String time});
}

/// @nodoc
class _$FamilyActionCopyWithImpl<$Res, $Val extends FamilyAction>
    implements $FamilyActionCopyWith<$Res> {
  _$FamilyActionCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of FamilyAction
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? text = null, Object? icon = null, Object? time = null}) {
    return _then(
      _value.copyWith(
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
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$FamilyActionImplCopyWith<$Res>
    implements $FamilyActionCopyWith<$Res> {
  factory _$$FamilyActionImplCopyWith(
    _$FamilyActionImpl value,
    $Res Function(_$FamilyActionImpl) then,
  ) = __$$FamilyActionImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String text, String icon, String time});
}

/// @nodoc
class __$$FamilyActionImplCopyWithImpl<$Res>
    extends _$FamilyActionCopyWithImpl<$Res, _$FamilyActionImpl>
    implements _$$FamilyActionImplCopyWith<$Res> {
  __$$FamilyActionImplCopyWithImpl(
    _$FamilyActionImpl _value,
    $Res Function(_$FamilyActionImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of FamilyAction
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? text = null, Object? icon = null, Object? time = null}) {
    return _then(
      _$FamilyActionImpl(
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
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$FamilyActionImpl implements _FamilyAction {
  const _$FamilyActionImpl({
    required this.text,
    required this.icon,
    required this.time,
  });

  factory _$FamilyActionImpl.fromJson(Map<String, dynamic> json) =>
      _$$FamilyActionImplFromJson(json);

  @override
  final String text;
  @override
  final String icon;
  @override
  final String time;

  @override
  String toString() {
    return 'FamilyAction(text: $text, icon: $icon, time: $time)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$FamilyActionImpl &&
            (identical(other.text, text) || other.text == text) &&
            (identical(other.icon, icon) || other.icon == icon) &&
            (identical(other.time, time) || other.time == time));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, text, icon, time);

  /// Create a copy of FamilyAction
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$FamilyActionImplCopyWith<_$FamilyActionImpl> get copyWith =>
      __$$FamilyActionImplCopyWithImpl<_$FamilyActionImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$FamilyActionImplToJson(this);
  }
}

abstract class _FamilyAction implements FamilyAction {
  const factory _FamilyAction({
    required final String text,
    required final String icon,
    required final String time,
  }) = _$FamilyActionImpl;

  factory _FamilyAction.fromJson(Map<String, dynamic> json) =
      _$FamilyActionImpl.fromJson;

  @override
  String get text;
  @override
  String get icon;
  @override
  String get time;

  /// Create a copy of FamilyAction
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$FamilyActionImplCopyWith<_$FamilyActionImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
