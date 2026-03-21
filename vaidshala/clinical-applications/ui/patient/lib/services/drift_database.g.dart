// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'drift_database.dart';

// ignore_for_file: type=lint
class $CheckinQueueTable extends CheckinQueue
    with TableInfo<$CheckinQueueTable, CheckinQueueData> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $CheckinQueueTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<int> id = GeneratedColumn<int>(
    'id',
    aliasedName,
    false,
    hasAutoIncrement: true,
    type: DriftSqlType.int,
    requiredDuringInsert: false,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'PRIMARY KEY AUTOINCREMENT',
    ),
  );
  static const VerificationMeta _actionIdMeta = const VerificationMeta(
    'actionId',
  );
  @override
  late final GeneratedColumn<String> actionId = GeneratedColumn<String>(
    'action_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _completedMeta = const VerificationMeta(
    'completed',
  );
  @override
  late final GeneratedColumn<bool> completed = GeneratedColumn<bool>(
    'completed',
    aliasedName,
    false,
    type: DriftSqlType.bool,
    requiredDuringInsert: true,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'CHECK ("completed" IN (0, 1))',
    ),
  );
  static const VerificationMeta _timestampMeta = const VerificationMeta(
    'timestamp',
  );
  @override
  late final GeneratedColumn<DateTime> timestamp = GeneratedColumn<DateTime>(
    'timestamp',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _syncedMeta = const VerificationMeta('synced');
  @override
  late final GeneratedColumn<bool> synced = GeneratedColumn<bool>(
    'synced',
    aliasedName,
    false,
    type: DriftSqlType.bool,
    requiredDuringInsert: false,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'CHECK ("synced" IN (0, 1))',
    ),
    defaultValue: const Constant(false),
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    actionId,
    completed,
    timestamp,
    synced,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'checkin_queue';
  @override
  VerificationContext validateIntegrity(
    Insertable<CheckinQueueData> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    }
    if (data.containsKey('action_id')) {
      context.handle(
        _actionIdMeta,
        actionId.isAcceptableOrUnknown(data['action_id']!, _actionIdMeta),
      );
    } else if (isInserting) {
      context.missing(_actionIdMeta);
    }
    if (data.containsKey('completed')) {
      context.handle(
        _completedMeta,
        completed.isAcceptableOrUnknown(data['completed']!, _completedMeta),
      );
    } else if (isInserting) {
      context.missing(_completedMeta);
    }
    if (data.containsKey('timestamp')) {
      context.handle(
        _timestampMeta,
        timestamp.isAcceptableOrUnknown(data['timestamp']!, _timestampMeta),
      );
    } else if (isInserting) {
      context.missing(_timestampMeta);
    }
    if (data.containsKey('synced')) {
      context.handle(
        _syncedMeta,
        synced.isAcceptableOrUnknown(data['synced']!, _syncedMeta),
      );
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  CheckinQueueData map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return CheckinQueueData(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}id'],
      )!,
      actionId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}action_id'],
      )!,
      completed: attachedDatabase.typeMapping.read(
        DriftSqlType.bool,
        data['${effectivePrefix}completed'],
      )!,
      timestamp: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}timestamp'],
      )!,
      synced: attachedDatabase.typeMapping.read(
        DriftSqlType.bool,
        data['${effectivePrefix}synced'],
      )!,
    );
  }

  @override
  $CheckinQueueTable createAlias(String alias) {
    return $CheckinQueueTable(attachedDatabase, alias);
  }
}

class CheckinQueueData extends DataClass
    implements Insertable<CheckinQueueData> {
  final int id;
  final String actionId;
  final bool completed;
  final DateTime timestamp;
  final bool synced;
  const CheckinQueueData({
    required this.id,
    required this.actionId,
    required this.completed,
    required this.timestamp,
    required this.synced,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<int>(id);
    map['action_id'] = Variable<String>(actionId);
    map['completed'] = Variable<bool>(completed);
    map['timestamp'] = Variable<DateTime>(timestamp);
    map['synced'] = Variable<bool>(synced);
    return map;
  }

  CheckinQueueCompanion toCompanion(bool nullToAbsent) {
    return CheckinQueueCompanion(
      id: Value(id),
      actionId: Value(actionId),
      completed: Value(completed),
      timestamp: Value(timestamp),
      synced: Value(synced),
    );
  }

  factory CheckinQueueData.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return CheckinQueueData(
      id: serializer.fromJson<int>(json['id']),
      actionId: serializer.fromJson<String>(json['actionId']),
      completed: serializer.fromJson<bool>(json['completed']),
      timestamp: serializer.fromJson<DateTime>(json['timestamp']),
      synced: serializer.fromJson<bool>(json['synced']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<int>(id),
      'actionId': serializer.toJson<String>(actionId),
      'completed': serializer.toJson<bool>(completed),
      'timestamp': serializer.toJson<DateTime>(timestamp),
      'synced': serializer.toJson<bool>(synced),
    };
  }

  CheckinQueueData copyWith({
    int? id,
    String? actionId,
    bool? completed,
    DateTime? timestamp,
    bool? synced,
  }) => CheckinQueueData(
    id: id ?? this.id,
    actionId: actionId ?? this.actionId,
    completed: completed ?? this.completed,
    timestamp: timestamp ?? this.timestamp,
    synced: synced ?? this.synced,
  );
  CheckinQueueData copyWithCompanion(CheckinQueueCompanion data) {
    return CheckinQueueData(
      id: data.id.present ? data.id.value : this.id,
      actionId: data.actionId.present ? data.actionId.value : this.actionId,
      completed: data.completed.present ? data.completed.value : this.completed,
      timestamp: data.timestamp.present ? data.timestamp.value : this.timestamp,
      synced: data.synced.present ? data.synced.value : this.synced,
    );
  }

  @override
  String toString() {
    return (StringBuffer('CheckinQueueData(')
          ..write('id: $id, ')
          ..write('actionId: $actionId, ')
          ..write('completed: $completed, ')
          ..write('timestamp: $timestamp, ')
          ..write('synced: $synced')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(id, actionId, completed, timestamp, synced);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CheckinQueueData &&
          other.id == this.id &&
          other.actionId == this.actionId &&
          other.completed == this.completed &&
          other.timestamp == this.timestamp &&
          other.synced == this.synced);
}

class CheckinQueueCompanion extends UpdateCompanion<CheckinQueueData> {
  final Value<int> id;
  final Value<String> actionId;
  final Value<bool> completed;
  final Value<DateTime> timestamp;
  final Value<bool> synced;
  const CheckinQueueCompanion({
    this.id = const Value.absent(),
    this.actionId = const Value.absent(),
    this.completed = const Value.absent(),
    this.timestamp = const Value.absent(),
    this.synced = const Value.absent(),
  });
  CheckinQueueCompanion.insert({
    this.id = const Value.absent(),
    required String actionId,
    required bool completed,
    required DateTime timestamp,
    this.synced = const Value.absent(),
  }) : actionId = Value(actionId),
       completed = Value(completed),
       timestamp = Value(timestamp);
  static Insertable<CheckinQueueData> custom({
    Expression<int>? id,
    Expression<String>? actionId,
    Expression<bool>? completed,
    Expression<DateTime>? timestamp,
    Expression<bool>? synced,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (actionId != null) 'action_id': actionId,
      if (completed != null) 'completed': completed,
      if (timestamp != null) 'timestamp': timestamp,
      if (synced != null) 'synced': synced,
    });
  }

  CheckinQueueCompanion copyWith({
    Value<int>? id,
    Value<String>? actionId,
    Value<bool>? completed,
    Value<DateTime>? timestamp,
    Value<bool>? synced,
  }) {
    return CheckinQueueCompanion(
      id: id ?? this.id,
      actionId: actionId ?? this.actionId,
      completed: completed ?? this.completed,
      timestamp: timestamp ?? this.timestamp,
      synced: synced ?? this.synced,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<int>(id.value);
    }
    if (actionId.present) {
      map['action_id'] = Variable<String>(actionId.value);
    }
    if (completed.present) {
      map['completed'] = Variable<bool>(completed.value);
    }
    if (timestamp.present) {
      map['timestamp'] = Variable<DateTime>(timestamp.value);
    }
    if (synced.present) {
      map['synced'] = Variable<bool>(synced.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('CheckinQueueCompanion(')
          ..write('id: $id, ')
          ..write('actionId: $actionId, ')
          ..write('completed: $completed, ')
          ..write('timestamp: $timestamp, ')
          ..write('synced: $synced')
          ..write(')'))
        .toString();
  }
}

class $LabHistoryTable extends LabHistory
    with TableInfo<$LabHistoryTable, LabHistoryData> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $LabHistoryTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<int> id = GeneratedColumn<int>(
    'id',
    aliasedName,
    false,
    hasAutoIncrement: true,
    type: DriftSqlType.int,
    requiredDuringInsert: false,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'PRIMARY KEY AUTOINCREMENT',
    ),
  );
  static const VerificationMeta _metricIdMeta = const VerificationMeta(
    'metricId',
  );
  @override
  late final GeneratedColumn<String> metricId = GeneratedColumn<String>(
    'metric_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _valueMeta = const VerificationMeta('value');
  @override
  late final GeneratedColumn<double> value = GeneratedColumn<double>(
    'value',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _unitMeta = const VerificationMeta('unit');
  @override
  late final GeneratedColumn<String> unit = GeneratedColumn<String>(
    'unit',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _recordedAtMeta = const VerificationMeta(
    'recordedAt',
  );
  @override
  late final GeneratedColumn<DateTime> recordedAt = GeneratedColumn<DateTime>(
    'recorded_at',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [id, metricId, value, unit, recordedAt];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'lab_history';
  @override
  VerificationContext validateIntegrity(
    Insertable<LabHistoryData> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    }
    if (data.containsKey('metric_id')) {
      context.handle(
        _metricIdMeta,
        metricId.isAcceptableOrUnknown(data['metric_id']!, _metricIdMeta),
      );
    } else if (isInserting) {
      context.missing(_metricIdMeta);
    }
    if (data.containsKey('value')) {
      context.handle(
        _valueMeta,
        value.isAcceptableOrUnknown(data['value']!, _valueMeta),
      );
    } else if (isInserting) {
      context.missing(_valueMeta);
    }
    if (data.containsKey('unit')) {
      context.handle(
        _unitMeta,
        unit.isAcceptableOrUnknown(data['unit']!, _unitMeta),
      );
    } else if (isInserting) {
      context.missing(_unitMeta);
    }
    if (data.containsKey('recorded_at')) {
      context.handle(
        _recordedAtMeta,
        recordedAt.isAcceptableOrUnknown(data['recorded_at']!, _recordedAtMeta),
      );
    } else if (isInserting) {
      context.missing(_recordedAtMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  LabHistoryData map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return LabHistoryData(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}id'],
      )!,
      metricId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}metric_id'],
      )!,
      value: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}value'],
      )!,
      unit: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}unit'],
      )!,
      recordedAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}recorded_at'],
      )!,
    );
  }

  @override
  $LabHistoryTable createAlias(String alias) {
    return $LabHistoryTable(attachedDatabase, alias);
  }
}

class LabHistoryData extends DataClass implements Insertable<LabHistoryData> {
  final int id;
  final String metricId;
  final double value;
  final String unit;
  final DateTime recordedAt;
  const LabHistoryData({
    required this.id,
    required this.metricId,
    required this.value,
    required this.unit,
    required this.recordedAt,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<int>(id);
    map['metric_id'] = Variable<String>(metricId);
    map['value'] = Variable<double>(value);
    map['unit'] = Variable<String>(unit);
    map['recorded_at'] = Variable<DateTime>(recordedAt);
    return map;
  }

  LabHistoryCompanion toCompanion(bool nullToAbsent) {
    return LabHistoryCompanion(
      id: Value(id),
      metricId: Value(metricId),
      value: Value(value),
      unit: Value(unit),
      recordedAt: Value(recordedAt),
    );
  }

  factory LabHistoryData.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return LabHistoryData(
      id: serializer.fromJson<int>(json['id']),
      metricId: serializer.fromJson<String>(json['metricId']),
      value: serializer.fromJson<double>(json['value']),
      unit: serializer.fromJson<String>(json['unit']),
      recordedAt: serializer.fromJson<DateTime>(json['recordedAt']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<int>(id),
      'metricId': serializer.toJson<String>(metricId),
      'value': serializer.toJson<double>(value),
      'unit': serializer.toJson<String>(unit),
      'recordedAt': serializer.toJson<DateTime>(recordedAt),
    };
  }

  LabHistoryData copyWith({
    int? id,
    String? metricId,
    double? value,
    String? unit,
    DateTime? recordedAt,
  }) => LabHistoryData(
    id: id ?? this.id,
    metricId: metricId ?? this.metricId,
    value: value ?? this.value,
    unit: unit ?? this.unit,
    recordedAt: recordedAt ?? this.recordedAt,
  );
  LabHistoryData copyWithCompanion(LabHistoryCompanion data) {
    return LabHistoryData(
      id: data.id.present ? data.id.value : this.id,
      metricId: data.metricId.present ? data.metricId.value : this.metricId,
      value: data.value.present ? data.value.value : this.value,
      unit: data.unit.present ? data.unit.value : this.unit,
      recordedAt: data.recordedAt.present
          ? data.recordedAt.value
          : this.recordedAt,
    );
  }

  @override
  String toString() {
    return (StringBuffer('LabHistoryData(')
          ..write('id: $id, ')
          ..write('metricId: $metricId, ')
          ..write('value: $value, ')
          ..write('unit: $unit, ')
          ..write('recordedAt: $recordedAt')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(id, metricId, value, unit, recordedAt);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is LabHistoryData &&
          other.id == this.id &&
          other.metricId == this.metricId &&
          other.value == this.value &&
          other.unit == this.unit &&
          other.recordedAt == this.recordedAt);
}

class LabHistoryCompanion extends UpdateCompanion<LabHistoryData> {
  final Value<int> id;
  final Value<String> metricId;
  final Value<double> value;
  final Value<String> unit;
  final Value<DateTime> recordedAt;
  const LabHistoryCompanion({
    this.id = const Value.absent(),
    this.metricId = const Value.absent(),
    this.value = const Value.absent(),
    this.unit = const Value.absent(),
    this.recordedAt = const Value.absent(),
  });
  LabHistoryCompanion.insert({
    this.id = const Value.absent(),
    required String metricId,
    required double value,
    required String unit,
    required DateTime recordedAt,
  }) : metricId = Value(metricId),
       value = Value(value),
       unit = Value(unit),
       recordedAt = Value(recordedAt);
  static Insertable<LabHistoryData> custom({
    Expression<int>? id,
    Expression<String>? metricId,
    Expression<double>? value,
    Expression<String>? unit,
    Expression<DateTime>? recordedAt,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (metricId != null) 'metric_id': metricId,
      if (value != null) 'value': value,
      if (unit != null) 'unit': unit,
      if (recordedAt != null) 'recorded_at': recordedAt,
    });
  }

  LabHistoryCompanion copyWith({
    Value<int>? id,
    Value<String>? metricId,
    Value<double>? value,
    Value<String>? unit,
    Value<DateTime>? recordedAt,
  }) {
    return LabHistoryCompanion(
      id: id ?? this.id,
      metricId: metricId ?? this.metricId,
      value: value ?? this.value,
      unit: unit ?? this.unit,
      recordedAt: recordedAt ?? this.recordedAt,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<int>(id.value);
    }
    if (metricId.present) {
      map['metric_id'] = Variable<String>(metricId.value);
    }
    if (value.present) {
      map['value'] = Variable<double>(value.value);
    }
    if (unit.present) {
      map['unit'] = Variable<String>(unit.value);
    }
    if (recordedAt.present) {
      map['recorded_at'] = Variable<DateTime>(recordedAt.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('LabHistoryCompanion(')
          ..write('id: $id, ')
          ..write('metricId: $metricId, ')
          ..write('value: $value, ')
          ..write('unit: $unit, ')
          ..write('recordedAt: $recordedAt')
          ..write(')'))
        .toString();
  }
}

class $NotificationsTable extends Notifications
    with TableInfo<$NotificationsTable, Notification> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $NotificationsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _typeMeta = const VerificationMeta('type');
  @override
  late final GeneratedColumn<String> type = GeneratedColumn<String>(
    'type',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _titleMeta = const VerificationMeta('title');
  @override
  late final GeneratedColumn<String> title = GeneratedColumn<String>(
    'title',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _bodyMeta = const VerificationMeta('body');
  @override
  late final GeneratedColumn<String> body = GeneratedColumn<String>(
    'body',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _deepLinkMeta = const VerificationMeta(
    'deepLink',
  );
  @override
  late final GeneratedColumn<String> deepLink = GeneratedColumn<String>(
    'deep_link',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _timestampMeta = const VerificationMeta(
    'timestamp',
  );
  @override
  late final GeneratedColumn<int> timestamp = GeneratedColumn<int>(
    'timestamp',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _readMeta = const VerificationMeta('read');
  @override
  late final GeneratedColumn<bool> read = GeneratedColumn<bool>(
    'read',
    aliasedName,
    false,
    type: DriftSqlType.bool,
    requiredDuringInsert: false,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'CHECK ("read" IN (0, 1))',
    ),
    defaultValue: const Constant(false),
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    type,
    title,
    body,
    deepLink,
    timestamp,
    read,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'notifications';
  @override
  VerificationContext validateIntegrity(
    Insertable<Notification> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('type')) {
      context.handle(
        _typeMeta,
        type.isAcceptableOrUnknown(data['type']!, _typeMeta),
      );
    } else if (isInserting) {
      context.missing(_typeMeta);
    }
    if (data.containsKey('title')) {
      context.handle(
        _titleMeta,
        title.isAcceptableOrUnknown(data['title']!, _titleMeta),
      );
    } else if (isInserting) {
      context.missing(_titleMeta);
    }
    if (data.containsKey('body')) {
      context.handle(
        _bodyMeta,
        body.isAcceptableOrUnknown(data['body']!, _bodyMeta),
      );
    } else if (isInserting) {
      context.missing(_bodyMeta);
    }
    if (data.containsKey('deep_link')) {
      context.handle(
        _deepLinkMeta,
        deepLink.isAcceptableOrUnknown(data['deep_link']!, _deepLinkMeta),
      );
    }
    if (data.containsKey('timestamp')) {
      context.handle(
        _timestampMeta,
        timestamp.isAcceptableOrUnknown(data['timestamp']!, _timestampMeta),
      );
    } else if (isInserting) {
      context.missing(_timestampMeta);
    }
    if (data.containsKey('read')) {
      context.handle(
        _readMeta,
        read.isAcceptableOrUnknown(data['read']!, _readMeta),
      );
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  Notification map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return Notification(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      type: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}type'],
      )!,
      title: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}title'],
      )!,
      body: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}body'],
      )!,
      deepLink: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}deep_link'],
      ),
      timestamp: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}timestamp'],
      )!,
      read: attachedDatabase.typeMapping.read(
        DriftSqlType.bool,
        data['${effectivePrefix}read'],
      )!,
    );
  }

  @override
  $NotificationsTable createAlias(String alias) {
    return $NotificationsTable(attachedDatabase, alias);
  }
}

class Notification extends DataClass implements Insertable<Notification> {
  final String id;
  final String type;
  final String title;
  final String body;
  final String? deepLink;
  final int timestamp;
  final bool read;
  const Notification({
    required this.id,
    required this.type,
    required this.title,
    required this.body,
    this.deepLink,
    required this.timestamp,
    required this.read,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['type'] = Variable<String>(type);
    map['title'] = Variable<String>(title);
    map['body'] = Variable<String>(body);
    if (!nullToAbsent || deepLink != null) {
      map['deep_link'] = Variable<String>(deepLink);
    }
    map['timestamp'] = Variable<int>(timestamp);
    map['read'] = Variable<bool>(read);
    return map;
  }

  NotificationsCompanion toCompanion(bool nullToAbsent) {
    return NotificationsCompanion(
      id: Value(id),
      type: Value(type),
      title: Value(title),
      body: Value(body),
      deepLink: deepLink == null && nullToAbsent
          ? const Value.absent()
          : Value(deepLink),
      timestamp: Value(timestamp),
      read: Value(read),
    );
  }

  factory Notification.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return Notification(
      id: serializer.fromJson<String>(json['id']),
      type: serializer.fromJson<String>(json['type']),
      title: serializer.fromJson<String>(json['title']),
      body: serializer.fromJson<String>(json['body']),
      deepLink: serializer.fromJson<String?>(json['deepLink']),
      timestamp: serializer.fromJson<int>(json['timestamp']),
      read: serializer.fromJson<bool>(json['read']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'type': serializer.toJson<String>(type),
      'title': serializer.toJson<String>(title),
      'body': serializer.toJson<String>(body),
      'deepLink': serializer.toJson<String?>(deepLink),
      'timestamp': serializer.toJson<int>(timestamp),
      'read': serializer.toJson<bool>(read),
    };
  }

  Notification copyWith({
    String? id,
    String? type,
    String? title,
    String? body,
    Value<String?> deepLink = const Value.absent(),
    int? timestamp,
    bool? read,
  }) => Notification(
    id: id ?? this.id,
    type: type ?? this.type,
    title: title ?? this.title,
    body: body ?? this.body,
    deepLink: deepLink.present ? deepLink.value : this.deepLink,
    timestamp: timestamp ?? this.timestamp,
    read: read ?? this.read,
  );
  Notification copyWithCompanion(NotificationsCompanion data) {
    return Notification(
      id: data.id.present ? data.id.value : this.id,
      type: data.type.present ? data.type.value : this.type,
      title: data.title.present ? data.title.value : this.title,
      body: data.body.present ? data.body.value : this.body,
      deepLink: data.deepLink.present ? data.deepLink.value : this.deepLink,
      timestamp: data.timestamp.present ? data.timestamp.value : this.timestamp,
      read: data.read.present ? data.read.value : this.read,
    );
  }

  @override
  String toString() {
    return (StringBuffer('Notification(')
          ..write('id: $id, ')
          ..write('type: $type, ')
          ..write('title: $title, ')
          ..write('body: $body, ')
          ..write('deepLink: $deepLink, ')
          ..write('timestamp: $timestamp, ')
          ..write('read: $read')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode =>
      Object.hash(id, type, title, body, deepLink, timestamp, read);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is Notification &&
          other.id == this.id &&
          other.type == this.type &&
          other.title == this.title &&
          other.body == this.body &&
          other.deepLink == this.deepLink &&
          other.timestamp == this.timestamp &&
          other.read == this.read);
}

class NotificationsCompanion extends UpdateCompanion<Notification> {
  final Value<String> id;
  final Value<String> type;
  final Value<String> title;
  final Value<String> body;
  final Value<String?> deepLink;
  final Value<int> timestamp;
  final Value<bool> read;
  final Value<int> rowid;
  const NotificationsCompanion({
    this.id = const Value.absent(),
    this.type = const Value.absent(),
    this.title = const Value.absent(),
    this.body = const Value.absent(),
    this.deepLink = const Value.absent(),
    this.timestamp = const Value.absent(),
    this.read = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  NotificationsCompanion.insert({
    required String id,
    required String type,
    required String title,
    required String body,
    this.deepLink = const Value.absent(),
    required int timestamp,
    this.read = const Value.absent(),
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       type = Value(type),
       title = Value(title),
       body = Value(body),
       timestamp = Value(timestamp);
  static Insertable<Notification> custom({
    Expression<String>? id,
    Expression<String>? type,
    Expression<String>? title,
    Expression<String>? body,
    Expression<String>? deepLink,
    Expression<int>? timestamp,
    Expression<bool>? read,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (type != null) 'type': type,
      if (title != null) 'title': title,
      if (body != null) 'body': body,
      if (deepLink != null) 'deep_link': deepLink,
      if (timestamp != null) 'timestamp': timestamp,
      if (read != null) 'read': read,
      if (rowid != null) 'rowid': rowid,
    });
  }

  NotificationsCompanion copyWith({
    Value<String>? id,
    Value<String>? type,
    Value<String>? title,
    Value<String>? body,
    Value<String?>? deepLink,
    Value<int>? timestamp,
    Value<bool>? read,
    Value<int>? rowid,
  }) {
    return NotificationsCompanion(
      id: id ?? this.id,
      type: type ?? this.type,
      title: title ?? this.title,
      body: body ?? this.body,
      deepLink: deepLink ?? this.deepLink,
      timestamp: timestamp ?? this.timestamp,
      read: read ?? this.read,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (type.present) {
      map['type'] = Variable<String>(type.value);
    }
    if (title.present) {
      map['title'] = Variable<String>(title.value);
    }
    if (body.present) {
      map['body'] = Variable<String>(body.value);
    }
    if (deepLink.present) {
      map['deep_link'] = Variable<String>(deepLink.value);
    }
    if (timestamp.present) {
      map['timestamp'] = Variable<int>(timestamp.value);
    }
    if (read.present) {
      map['read'] = Variable<bool>(read.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('NotificationsCompanion(')
          ..write('id: $id, ')
          ..write('type: $type, ')
          ..write('title: $title, ')
          ..write('body: $body, ')
          ..write('deepLink: $deepLink, ')
          ..write('timestamp: $timestamp, ')
          ..write('read: $read, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $ObservationQueueTable extends ObservationQueue
    with TableInfo<$ObservationQueueTable, ObservationQueueData> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $ObservationQueueTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _typeMeta = const VerificationMeta('type');
  @override
  late final GeneratedColumn<String> type = GeneratedColumn<String>(
    'type',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _valueMeta = const VerificationMeta('value');
  @override
  late final GeneratedColumn<String> value = GeneratedColumn<String>(
    'value',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _unitMeta = const VerificationMeta('unit');
  @override
  late final GeneratedColumn<String> unit = GeneratedColumn<String>(
    'unit',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _timestampMeta = const VerificationMeta(
    'timestamp',
  );
  @override
  late final GeneratedColumn<int> timestamp = GeneratedColumn<int>(
    'timestamp',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _syncedMeta = const VerificationMeta('synced');
  @override
  late final GeneratedColumn<bool> synced = GeneratedColumn<bool>(
    'synced',
    aliasedName,
    false,
    type: DriftSqlType.bool,
    requiredDuringInsert: false,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'CHECK ("synced" IN (0, 1))',
    ),
    defaultValue: const Constant(false),
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    type,
    value,
    unit,
    timestamp,
    synced,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'observation_queue';
  @override
  VerificationContext validateIntegrity(
    Insertable<ObservationQueueData> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('type')) {
      context.handle(
        _typeMeta,
        type.isAcceptableOrUnknown(data['type']!, _typeMeta),
      );
    } else if (isInserting) {
      context.missing(_typeMeta);
    }
    if (data.containsKey('value')) {
      context.handle(
        _valueMeta,
        value.isAcceptableOrUnknown(data['value']!, _valueMeta),
      );
    } else if (isInserting) {
      context.missing(_valueMeta);
    }
    if (data.containsKey('unit')) {
      context.handle(
        _unitMeta,
        unit.isAcceptableOrUnknown(data['unit']!, _unitMeta),
      );
    } else if (isInserting) {
      context.missing(_unitMeta);
    }
    if (data.containsKey('timestamp')) {
      context.handle(
        _timestampMeta,
        timestamp.isAcceptableOrUnknown(data['timestamp']!, _timestampMeta),
      );
    } else if (isInserting) {
      context.missing(_timestampMeta);
    }
    if (data.containsKey('synced')) {
      context.handle(
        _syncedMeta,
        synced.isAcceptableOrUnknown(data['synced']!, _syncedMeta),
      );
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  ObservationQueueData map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return ObservationQueueData(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      type: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}type'],
      )!,
      value: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}value'],
      )!,
      unit: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}unit'],
      )!,
      timestamp: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}timestamp'],
      )!,
      synced: attachedDatabase.typeMapping.read(
        DriftSqlType.bool,
        data['${effectivePrefix}synced'],
      )!,
    );
  }

  @override
  $ObservationQueueTable createAlias(String alias) {
    return $ObservationQueueTable(attachedDatabase, alias);
  }
}

class ObservationQueueData extends DataClass
    implements Insertable<ObservationQueueData> {
  final String id;
  final String type;
  final String value;
  final String unit;
  final int timestamp;
  final bool synced;
  const ObservationQueueData({
    required this.id,
    required this.type,
    required this.value,
    required this.unit,
    required this.timestamp,
    required this.synced,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['type'] = Variable<String>(type);
    map['value'] = Variable<String>(value);
    map['unit'] = Variable<String>(unit);
    map['timestamp'] = Variable<int>(timestamp);
    map['synced'] = Variable<bool>(synced);
    return map;
  }

  ObservationQueueCompanion toCompanion(bool nullToAbsent) {
    return ObservationQueueCompanion(
      id: Value(id),
      type: Value(type),
      value: Value(value),
      unit: Value(unit),
      timestamp: Value(timestamp),
      synced: Value(synced),
    );
  }

  factory ObservationQueueData.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return ObservationQueueData(
      id: serializer.fromJson<String>(json['id']),
      type: serializer.fromJson<String>(json['type']),
      value: serializer.fromJson<String>(json['value']),
      unit: serializer.fromJson<String>(json['unit']),
      timestamp: serializer.fromJson<int>(json['timestamp']),
      synced: serializer.fromJson<bool>(json['synced']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'type': serializer.toJson<String>(type),
      'value': serializer.toJson<String>(value),
      'unit': serializer.toJson<String>(unit),
      'timestamp': serializer.toJson<int>(timestamp),
      'synced': serializer.toJson<bool>(synced),
    };
  }

  ObservationQueueData copyWith({
    String? id,
    String? type,
    String? value,
    String? unit,
    int? timestamp,
    bool? synced,
  }) => ObservationQueueData(
    id: id ?? this.id,
    type: type ?? this.type,
    value: value ?? this.value,
    unit: unit ?? this.unit,
    timestamp: timestamp ?? this.timestamp,
    synced: synced ?? this.synced,
  );
  ObservationQueueData copyWithCompanion(ObservationQueueCompanion data) {
    return ObservationQueueData(
      id: data.id.present ? data.id.value : this.id,
      type: data.type.present ? data.type.value : this.type,
      value: data.value.present ? data.value.value : this.value,
      unit: data.unit.present ? data.unit.value : this.unit,
      timestamp: data.timestamp.present ? data.timestamp.value : this.timestamp,
      synced: data.synced.present ? data.synced.value : this.synced,
    );
  }

  @override
  String toString() {
    return (StringBuffer('ObservationQueueData(')
          ..write('id: $id, ')
          ..write('type: $type, ')
          ..write('value: $value, ')
          ..write('unit: $unit, ')
          ..write('timestamp: $timestamp, ')
          ..write('synced: $synced')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(id, type, value, unit, timestamp, synced);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is ObservationQueueData &&
          other.id == this.id &&
          other.type == this.type &&
          other.value == this.value &&
          other.unit == this.unit &&
          other.timestamp == this.timestamp &&
          other.synced == this.synced);
}

class ObservationQueueCompanion extends UpdateCompanion<ObservationQueueData> {
  final Value<String> id;
  final Value<String> type;
  final Value<String> value;
  final Value<String> unit;
  final Value<int> timestamp;
  final Value<bool> synced;
  final Value<int> rowid;
  const ObservationQueueCompanion({
    this.id = const Value.absent(),
    this.type = const Value.absent(),
    this.value = const Value.absent(),
    this.unit = const Value.absent(),
    this.timestamp = const Value.absent(),
    this.synced = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  ObservationQueueCompanion.insert({
    required String id,
    required String type,
    required String value,
    required String unit,
    required int timestamp,
    this.synced = const Value.absent(),
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       type = Value(type),
       value = Value(value),
       unit = Value(unit),
       timestamp = Value(timestamp);
  static Insertable<ObservationQueueData> custom({
    Expression<String>? id,
    Expression<String>? type,
    Expression<String>? value,
    Expression<String>? unit,
    Expression<int>? timestamp,
    Expression<bool>? synced,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (type != null) 'type': type,
      if (value != null) 'value': value,
      if (unit != null) 'unit': unit,
      if (timestamp != null) 'timestamp': timestamp,
      if (synced != null) 'synced': synced,
      if (rowid != null) 'rowid': rowid,
    });
  }

  ObservationQueueCompanion copyWith({
    Value<String>? id,
    Value<String>? type,
    Value<String>? value,
    Value<String>? unit,
    Value<int>? timestamp,
    Value<bool>? synced,
    Value<int>? rowid,
  }) {
    return ObservationQueueCompanion(
      id: id ?? this.id,
      type: type ?? this.type,
      value: value ?? this.value,
      unit: unit ?? this.unit,
      timestamp: timestamp ?? this.timestamp,
      synced: synced ?? this.synced,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (type.present) {
      map['type'] = Variable<String>(type.value);
    }
    if (value.present) {
      map['value'] = Variable<String>(value.value);
    }
    if (unit.present) {
      map['unit'] = Variable<String>(unit.value);
    }
    if (timestamp.present) {
      map['timestamp'] = Variable<int>(timestamp.value);
    }
    if (synced.present) {
      map['synced'] = Variable<bool>(synced.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('ObservationQueueCompanion(')
          ..write('id: $id, ')
          ..write('type: $type, ')
          ..write('value: $value, ')
          ..write('unit: $unit, ')
          ..write('timestamp: $timestamp, ')
          ..write('synced: $synced, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $MedicationLogTable extends MedicationLog
    with TableInfo<$MedicationLogTable, MedicationLogData> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $MedicationLogTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _actionIdMeta = const VerificationMeta(
    'actionId',
  );
  @override
  late final GeneratedColumn<String> actionId = GeneratedColumn<String>(
    'action_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _medicationNameMeta = const VerificationMeta(
    'medicationName',
  );
  @override
  late final GeneratedColumn<String> medicationName = GeneratedColumn<String>(
    'medication_name',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _completedMeta = const VerificationMeta(
    'completed',
  );
  @override
  late final GeneratedColumn<bool> completed = GeneratedColumn<bool>(
    'completed',
    aliasedName,
    false,
    type: DriftSqlType.bool,
    requiredDuringInsert: true,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'CHECK ("completed" IN (0, 1))',
    ),
  );
  static const VerificationMeta _timestampMeta = const VerificationMeta(
    'timestamp',
  );
  @override
  late final GeneratedColumn<int> timestamp = GeneratedColumn<int>(
    'timestamp',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    actionId,
    medicationName,
    completed,
    timestamp,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'medication_log';
  @override
  VerificationContext validateIntegrity(
    Insertable<MedicationLogData> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('action_id')) {
      context.handle(
        _actionIdMeta,
        actionId.isAcceptableOrUnknown(data['action_id']!, _actionIdMeta),
      );
    } else if (isInserting) {
      context.missing(_actionIdMeta);
    }
    if (data.containsKey('medication_name')) {
      context.handle(
        _medicationNameMeta,
        medicationName.isAcceptableOrUnknown(
          data['medication_name']!,
          _medicationNameMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_medicationNameMeta);
    }
    if (data.containsKey('completed')) {
      context.handle(
        _completedMeta,
        completed.isAcceptableOrUnknown(data['completed']!, _completedMeta),
      );
    } else if (isInserting) {
      context.missing(_completedMeta);
    }
    if (data.containsKey('timestamp')) {
      context.handle(
        _timestampMeta,
        timestamp.isAcceptableOrUnknown(data['timestamp']!, _timestampMeta),
      );
    } else if (isInserting) {
      context.missing(_timestampMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  MedicationLogData map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return MedicationLogData(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      actionId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}action_id'],
      )!,
      medicationName: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}medication_name'],
      )!,
      completed: attachedDatabase.typeMapping.read(
        DriftSqlType.bool,
        data['${effectivePrefix}completed'],
      )!,
      timestamp: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}timestamp'],
      )!,
    );
  }

  @override
  $MedicationLogTable createAlias(String alias) {
    return $MedicationLogTable(attachedDatabase, alias);
  }
}

class MedicationLogData extends DataClass
    implements Insertable<MedicationLogData> {
  final String id;
  final String actionId;
  final String medicationName;
  final bool completed;
  final int timestamp;
  const MedicationLogData({
    required this.id,
    required this.actionId,
    required this.medicationName,
    required this.completed,
    required this.timestamp,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['action_id'] = Variable<String>(actionId);
    map['medication_name'] = Variable<String>(medicationName);
    map['completed'] = Variable<bool>(completed);
    map['timestamp'] = Variable<int>(timestamp);
    return map;
  }

  MedicationLogCompanion toCompanion(bool nullToAbsent) {
    return MedicationLogCompanion(
      id: Value(id),
      actionId: Value(actionId),
      medicationName: Value(medicationName),
      completed: Value(completed),
      timestamp: Value(timestamp),
    );
  }

  factory MedicationLogData.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return MedicationLogData(
      id: serializer.fromJson<String>(json['id']),
      actionId: serializer.fromJson<String>(json['actionId']),
      medicationName: serializer.fromJson<String>(json['medicationName']),
      completed: serializer.fromJson<bool>(json['completed']),
      timestamp: serializer.fromJson<int>(json['timestamp']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'actionId': serializer.toJson<String>(actionId),
      'medicationName': serializer.toJson<String>(medicationName),
      'completed': serializer.toJson<bool>(completed),
      'timestamp': serializer.toJson<int>(timestamp),
    };
  }

  MedicationLogData copyWith({
    String? id,
    String? actionId,
    String? medicationName,
    bool? completed,
    int? timestamp,
  }) => MedicationLogData(
    id: id ?? this.id,
    actionId: actionId ?? this.actionId,
    medicationName: medicationName ?? this.medicationName,
    completed: completed ?? this.completed,
    timestamp: timestamp ?? this.timestamp,
  );
  MedicationLogData copyWithCompanion(MedicationLogCompanion data) {
    return MedicationLogData(
      id: data.id.present ? data.id.value : this.id,
      actionId: data.actionId.present ? data.actionId.value : this.actionId,
      medicationName: data.medicationName.present
          ? data.medicationName.value
          : this.medicationName,
      completed: data.completed.present ? data.completed.value : this.completed,
      timestamp: data.timestamp.present ? data.timestamp.value : this.timestamp,
    );
  }

  @override
  String toString() {
    return (StringBuffer('MedicationLogData(')
          ..write('id: $id, ')
          ..write('actionId: $actionId, ')
          ..write('medicationName: $medicationName, ')
          ..write('completed: $completed, ')
          ..write('timestamp: $timestamp')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode =>
      Object.hash(id, actionId, medicationName, completed, timestamp);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is MedicationLogData &&
          other.id == this.id &&
          other.actionId == this.actionId &&
          other.medicationName == this.medicationName &&
          other.completed == this.completed &&
          other.timestamp == this.timestamp);
}

class MedicationLogCompanion extends UpdateCompanion<MedicationLogData> {
  final Value<String> id;
  final Value<String> actionId;
  final Value<String> medicationName;
  final Value<bool> completed;
  final Value<int> timestamp;
  final Value<int> rowid;
  const MedicationLogCompanion({
    this.id = const Value.absent(),
    this.actionId = const Value.absent(),
    this.medicationName = const Value.absent(),
    this.completed = const Value.absent(),
    this.timestamp = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  MedicationLogCompanion.insert({
    required String id,
    required String actionId,
    required String medicationName,
    required bool completed,
    required int timestamp,
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       actionId = Value(actionId),
       medicationName = Value(medicationName),
       completed = Value(completed),
       timestamp = Value(timestamp);
  static Insertable<MedicationLogData> custom({
    Expression<String>? id,
    Expression<String>? actionId,
    Expression<String>? medicationName,
    Expression<bool>? completed,
    Expression<int>? timestamp,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (actionId != null) 'action_id': actionId,
      if (medicationName != null) 'medication_name': medicationName,
      if (completed != null) 'completed': completed,
      if (timestamp != null) 'timestamp': timestamp,
      if (rowid != null) 'rowid': rowid,
    });
  }

  MedicationLogCompanion copyWith({
    Value<String>? id,
    Value<String>? actionId,
    Value<String>? medicationName,
    Value<bool>? completed,
    Value<int>? timestamp,
    Value<int>? rowid,
  }) {
    return MedicationLogCompanion(
      id: id ?? this.id,
      actionId: actionId ?? this.actionId,
      medicationName: medicationName ?? this.medicationName,
      completed: completed ?? this.completed,
      timestamp: timestamp ?? this.timestamp,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (actionId.present) {
      map['action_id'] = Variable<String>(actionId.value);
    }
    if (medicationName.present) {
      map['medication_name'] = Variable<String>(medicationName.value);
    }
    if (completed.present) {
      map['completed'] = Variable<bool>(completed.value);
    }
    if (timestamp.present) {
      map['timestamp'] = Variable<int>(timestamp.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('MedicationLogCompanion(')
          ..write('id: $id, ')
          ..write('actionId: $actionId, ')
          ..write('medicationName: $medicationName, ')
          ..write('completed: $completed, ')
          ..write('timestamp: $timestamp, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $SymptomLogTable extends SymptomLog
    with TableInfo<$SymptomLogTable, SymptomLogData> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $SymptomLogTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _symptomMeta = const VerificationMeta(
    'symptom',
  );
  @override
  late final GeneratedColumn<String> symptom = GeneratedColumn<String>(
    'symptom',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _severityMeta = const VerificationMeta(
    'severity',
  );
  @override
  late final GeneratedColumn<String> severity = GeneratedColumn<String>(
    'severity',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _notesMeta = const VerificationMeta('notes');
  @override
  late final GeneratedColumn<String> notes = GeneratedColumn<String>(
    'notes',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _timestampMeta = const VerificationMeta(
    'timestamp',
  );
  @override
  late final GeneratedColumn<int> timestamp = GeneratedColumn<int>(
    'timestamp',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _syncedMeta = const VerificationMeta('synced');
  @override
  late final GeneratedColumn<bool> synced = GeneratedColumn<bool>(
    'synced',
    aliasedName,
    false,
    type: DriftSqlType.bool,
    requiredDuringInsert: false,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'CHECK ("synced" IN (0, 1))',
    ),
    defaultValue: const Constant(false),
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    symptom,
    severity,
    notes,
    timestamp,
    synced,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'symptom_log';
  @override
  VerificationContext validateIntegrity(
    Insertable<SymptomLogData> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('symptom')) {
      context.handle(
        _symptomMeta,
        symptom.isAcceptableOrUnknown(data['symptom']!, _symptomMeta),
      );
    } else if (isInserting) {
      context.missing(_symptomMeta);
    }
    if (data.containsKey('severity')) {
      context.handle(
        _severityMeta,
        severity.isAcceptableOrUnknown(data['severity']!, _severityMeta),
      );
    } else if (isInserting) {
      context.missing(_severityMeta);
    }
    if (data.containsKey('notes')) {
      context.handle(
        _notesMeta,
        notes.isAcceptableOrUnknown(data['notes']!, _notesMeta),
      );
    }
    if (data.containsKey('timestamp')) {
      context.handle(
        _timestampMeta,
        timestamp.isAcceptableOrUnknown(data['timestamp']!, _timestampMeta),
      );
    } else if (isInserting) {
      context.missing(_timestampMeta);
    }
    if (data.containsKey('synced')) {
      context.handle(
        _syncedMeta,
        synced.isAcceptableOrUnknown(data['synced']!, _syncedMeta),
      );
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  SymptomLogData map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return SymptomLogData(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      symptom: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}symptom'],
      )!,
      severity: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}severity'],
      )!,
      notes: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}notes'],
      ),
      timestamp: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}timestamp'],
      )!,
      synced: attachedDatabase.typeMapping.read(
        DriftSqlType.bool,
        data['${effectivePrefix}synced'],
      )!,
    );
  }

  @override
  $SymptomLogTable createAlias(String alias) {
    return $SymptomLogTable(attachedDatabase, alias);
  }
}

class SymptomLogData extends DataClass implements Insertable<SymptomLogData> {
  final String id;
  final String symptom;
  final String severity;
  final String? notes;
  final int timestamp;
  final bool synced;
  const SymptomLogData({
    required this.id,
    required this.symptom,
    required this.severity,
    this.notes,
    required this.timestamp,
    required this.synced,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['symptom'] = Variable<String>(symptom);
    map['severity'] = Variable<String>(severity);
    if (!nullToAbsent || notes != null) {
      map['notes'] = Variable<String>(notes);
    }
    map['timestamp'] = Variable<int>(timestamp);
    map['synced'] = Variable<bool>(synced);
    return map;
  }

  SymptomLogCompanion toCompanion(bool nullToAbsent) {
    return SymptomLogCompanion(
      id: Value(id),
      symptom: Value(symptom),
      severity: Value(severity),
      notes: notes == null && nullToAbsent
          ? const Value.absent()
          : Value(notes),
      timestamp: Value(timestamp),
      synced: Value(synced),
    );
  }

  factory SymptomLogData.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return SymptomLogData(
      id: serializer.fromJson<String>(json['id']),
      symptom: serializer.fromJson<String>(json['symptom']),
      severity: serializer.fromJson<String>(json['severity']),
      notes: serializer.fromJson<String?>(json['notes']),
      timestamp: serializer.fromJson<int>(json['timestamp']),
      synced: serializer.fromJson<bool>(json['synced']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'symptom': serializer.toJson<String>(symptom),
      'severity': serializer.toJson<String>(severity),
      'notes': serializer.toJson<String?>(notes),
      'timestamp': serializer.toJson<int>(timestamp),
      'synced': serializer.toJson<bool>(synced),
    };
  }

  SymptomLogData copyWith({
    String? id,
    String? symptom,
    String? severity,
    Value<String?> notes = const Value.absent(),
    int? timestamp,
    bool? synced,
  }) => SymptomLogData(
    id: id ?? this.id,
    symptom: symptom ?? this.symptom,
    severity: severity ?? this.severity,
    notes: notes.present ? notes.value : this.notes,
    timestamp: timestamp ?? this.timestamp,
    synced: synced ?? this.synced,
  );
  SymptomLogData copyWithCompanion(SymptomLogCompanion data) {
    return SymptomLogData(
      id: data.id.present ? data.id.value : this.id,
      symptom: data.symptom.present ? data.symptom.value : this.symptom,
      severity: data.severity.present ? data.severity.value : this.severity,
      notes: data.notes.present ? data.notes.value : this.notes,
      timestamp: data.timestamp.present ? data.timestamp.value : this.timestamp,
      synced: data.synced.present ? data.synced.value : this.synced,
    );
  }

  @override
  String toString() {
    return (StringBuffer('SymptomLogData(')
          ..write('id: $id, ')
          ..write('symptom: $symptom, ')
          ..write('severity: $severity, ')
          ..write('notes: $notes, ')
          ..write('timestamp: $timestamp, ')
          ..write('synced: $synced')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode =>
      Object.hash(id, symptom, severity, notes, timestamp, synced);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is SymptomLogData &&
          other.id == this.id &&
          other.symptom == this.symptom &&
          other.severity == this.severity &&
          other.notes == this.notes &&
          other.timestamp == this.timestamp &&
          other.synced == this.synced);
}

class SymptomLogCompanion extends UpdateCompanion<SymptomLogData> {
  final Value<String> id;
  final Value<String> symptom;
  final Value<String> severity;
  final Value<String?> notes;
  final Value<int> timestamp;
  final Value<bool> synced;
  final Value<int> rowid;
  const SymptomLogCompanion({
    this.id = const Value.absent(),
    this.symptom = const Value.absent(),
    this.severity = const Value.absent(),
    this.notes = const Value.absent(),
    this.timestamp = const Value.absent(),
    this.synced = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  SymptomLogCompanion.insert({
    required String id,
    required String symptom,
    required String severity,
    this.notes = const Value.absent(),
    required int timestamp,
    this.synced = const Value.absent(),
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       symptom = Value(symptom),
       severity = Value(severity),
       timestamp = Value(timestamp);
  static Insertable<SymptomLogData> custom({
    Expression<String>? id,
    Expression<String>? symptom,
    Expression<String>? severity,
    Expression<String>? notes,
    Expression<int>? timestamp,
    Expression<bool>? synced,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (symptom != null) 'symptom': symptom,
      if (severity != null) 'severity': severity,
      if (notes != null) 'notes': notes,
      if (timestamp != null) 'timestamp': timestamp,
      if (synced != null) 'synced': synced,
      if (rowid != null) 'rowid': rowid,
    });
  }

  SymptomLogCompanion copyWith({
    Value<String>? id,
    Value<String>? symptom,
    Value<String>? severity,
    Value<String?>? notes,
    Value<int>? timestamp,
    Value<bool>? synced,
    Value<int>? rowid,
  }) {
    return SymptomLogCompanion(
      id: id ?? this.id,
      symptom: symptom ?? this.symptom,
      severity: severity ?? this.severity,
      notes: notes ?? this.notes,
      timestamp: timestamp ?? this.timestamp,
      synced: synced ?? this.synced,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (symptom.present) {
      map['symptom'] = Variable<String>(symptom.value);
    }
    if (severity.present) {
      map['severity'] = Variable<String>(severity.value);
    }
    if (notes.present) {
      map['notes'] = Variable<String>(notes.value);
    }
    if (timestamp.present) {
      map['timestamp'] = Variable<int>(timestamp.value);
    }
    if (synced.present) {
      map['synced'] = Variable<bool>(synced.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('SymptomLogCompanion(')
          ..write('id: $id, ')
          ..write('symptom: $symptom, ')
          ..write('severity: $severity, ')
          ..write('notes: $notes, ')
          ..write('timestamp: $timestamp, ')
          ..write('synced: $synced, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

abstract class _$AppDatabase extends GeneratedDatabase {
  _$AppDatabase(QueryExecutor e) : super(e);
  $AppDatabaseManager get managers => $AppDatabaseManager(this);
  late final $CheckinQueueTable checkinQueue = $CheckinQueueTable(this);
  late final $LabHistoryTable labHistory = $LabHistoryTable(this);
  late final $NotificationsTable notifications = $NotificationsTable(this);
  late final $ObservationQueueTable observationQueue = $ObservationQueueTable(
    this,
  );
  late final $MedicationLogTable medicationLog = $MedicationLogTable(this);
  late final $SymptomLogTable symptomLog = $SymptomLogTable(this);
  @override
  Iterable<TableInfo<Table, Object?>> get allTables =>
      allSchemaEntities.whereType<TableInfo<Table, Object?>>();
  @override
  List<DatabaseSchemaEntity> get allSchemaEntities => [
    checkinQueue,
    labHistory,
    notifications,
    observationQueue,
    medicationLog,
    symptomLog,
  ];
}

typedef $$CheckinQueueTableCreateCompanionBuilder =
    CheckinQueueCompanion Function({
      Value<int> id,
      required String actionId,
      required bool completed,
      required DateTime timestamp,
      Value<bool> synced,
    });
typedef $$CheckinQueueTableUpdateCompanionBuilder =
    CheckinQueueCompanion Function({
      Value<int> id,
      Value<String> actionId,
      Value<bool> completed,
      Value<DateTime> timestamp,
      Value<bool> synced,
    });

class $$CheckinQueueTableFilterComposer
    extends Composer<_$AppDatabase, $CheckinQueueTable> {
  $$CheckinQueueTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<int> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get actionId => $composableBuilder(
    column: $table.actionId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<bool> get completed => $composableBuilder(
    column: $table.completed,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<bool> get synced => $composableBuilder(
    column: $table.synced,
    builder: (column) => ColumnFilters(column),
  );
}

class $$CheckinQueueTableOrderingComposer
    extends Composer<_$AppDatabase, $CheckinQueueTable> {
  $$CheckinQueueTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<int> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get actionId => $composableBuilder(
    column: $table.actionId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<bool> get completed => $composableBuilder(
    column: $table.completed,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<bool> get synced => $composableBuilder(
    column: $table.synced,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$CheckinQueueTableAnnotationComposer
    extends Composer<_$AppDatabase, $CheckinQueueTable> {
  $$CheckinQueueTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<int> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get actionId =>
      $composableBuilder(column: $table.actionId, builder: (column) => column);

  GeneratedColumn<bool> get completed =>
      $composableBuilder(column: $table.completed, builder: (column) => column);

  GeneratedColumn<DateTime> get timestamp =>
      $composableBuilder(column: $table.timestamp, builder: (column) => column);

  GeneratedColumn<bool> get synced =>
      $composableBuilder(column: $table.synced, builder: (column) => column);
}

class $$CheckinQueueTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $CheckinQueueTable,
          CheckinQueueData,
          $$CheckinQueueTableFilterComposer,
          $$CheckinQueueTableOrderingComposer,
          $$CheckinQueueTableAnnotationComposer,
          $$CheckinQueueTableCreateCompanionBuilder,
          $$CheckinQueueTableUpdateCompanionBuilder,
          (
            CheckinQueueData,
            BaseReferences<_$AppDatabase, $CheckinQueueTable, CheckinQueueData>,
          ),
          CheckinQueueData,
          PrefetchHooks Function()
        > {
  $$CheckinQueueTableTableManager(_$AppDatabase db, $CheckinQueueTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$CheckinQueueTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$CheckinQueueTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$CheckinQueueTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<int> id = const Value.absent(),
                Value<String> actionId = const Value.absent(),
                Value<bool> completed = const Value.absent(),
                Value<DateTime> timestamp = const Value.absent(),
                Value<bool> synced = const Value.absent(),
              }) => CheckinQueueCompanion(
                id: id,
                actionId: actionId,
                completed: completed,
                timestamp: timestamp,
                synced: synced,
              ),
          createCompanionCallback:
              ({
                Value<int> id = const Value.absent(),
                required String actionId,
                required bool completed,
                required DateTime timestamp,
                Value<bool> synced = const Value.absent(),
              }) => CheckinQueueCompanion.insert(
                id: id,
                actionId: actionId,
                completed: completed,
                timestamp: timestamp,
                synced: synced,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$CheckinQueueTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $CheckinQueueTable,
      CheckinQueueData,
      $$CheckinQueueTableFilterComposer,
      $$CheckinQueueTableOrderingComposer,
      $$CheckinQueueTableAnnotationComposer,
      $$CheckinQueueTableCreateCompanionBuilder,
      $$CheckinQueueTableUpdateCompanionBuilder,
      (
        CheckinQueueData,
        BaseReferences<_$AppDatabase, $CheckinQueueTable, CheckinQueueData>,
      ),
      CheckinQueueData,
      PrefetchHooks Function()
    >;
typedef $$LabHistoryTableCreateCompanionBuilder =
    LabHistoryCompanion Function({
      Value<int> id,
      required String metricId,
      required double value,
      required String unit,
      required DateTime recordedAt,
    });
typedef $$LabHistoryTableUpdateCompanionBuilder =
    LabHistoryCompanion Function({
      Value<int> id,
      Value<String> metricId,
      Value<double> value,
      Value<String> unit,
      Value<DateTime> recordedAt,
    });

class $$LabHistoryTableFilterComposer
    extends Composer<_$AppDatabase, $LabHistoryTable> {
  $$LabHistoryTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<int> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get metricId => $composableBuilder(
    column: $table.metricId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get value => $composableBuilder(
    column: $table.value,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get unit => $composableBuilder(
    column: $table.unit,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get recordedAt => $composableBuilder(
    column: $table.recordedAt,
    builder: (column) => ColumnFilters(column),
  );
}

class $$LabHistoryTableOrderingComposer
    extends Composer<_$AppDatabase, $LabHistoryTable> {
  $$LabHistoryTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<int> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get metricId => $composableBuilder(
    column: $table.metricId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get value => $composableBuilder(
    column: $table.value,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get unit => $composableBuilder(
    column: $table.unit,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get recordedAt => $composableBuilder(
    column: $table.recordedAt,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$LabHistoryTableAnnotationComposer
    extends Composer<_$AppDatabase, $LabHistoryTable> {
  $$LabHistoryTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<int> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get metricId =>
      $composableBuilder(column: $table.metricId, builder: (column) => column);

  GeneratedColumn<double> get value =>
      $composableBuilder(column: $table.value, builder: (column) => column);

  GeneratedColumn<String> get unit =>
      $composableBuilder(column: $table.unit, builder: (column) => column);

  GeneratedColumn<DateTime> get recordedAt => $composableBuilder(
    column: $table.recordedAt,
    builder: (column) => column,
  );
}

class $$LabHistoryTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $LabHistoryTable,
          LabHistoryData,
          $$LabHistoryTableFilterComposer,
          $$LabHistoryTableOrderingComposer,
          $$LabHistoryTableAnnotationComposer,
          $$LabHistoryTableCreateCompanionBuilder,
          $$LabHistoryTableUpdateCompanionBuilder,
          (
            LabHistoryData,
            BaseReferences<_$AppDatabase, $LabHistoryTable, LabHistoryData>,
          ),
          LabHistoryData,
          PrefetchHooks Function()
        > {
  $$LabHistoryTableTableManager(_$AppDatabase db, $LabHistoryTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$LabHistoryTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$LabHistoryTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$LabHistoryTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<int> id = const Value.absent(),
                Value<String> metricId = const Value.absent(),
                Value<double> value = const Value.absent(),
                Value<String> unit = const Value.absent(),
                Value<DateTime> recordedAt = const Value.absent(),
              }) => LabHistoryCompanion(
                id: id,
                metricId: metricId,
                value: value,
                unit: unit,
                recordedAt: recordedAt,
              ),
          createCompanionCallback:
              ({
                Value<int> id = const Value.absent(),
                required String metricId,
                required double value,
                required String unit,
                required DateTime recordedAt,
              }) => LabHistoryCompanion.insert(
                id: id,
                metricId: metricId,
                value: value,
                unit: unit,
                recordedAt: recordedAt,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$LabHistoryTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $LabHistoryTable,
      LabHistoryData,
      $$LabHistoryTableFilterComposer,
      $$LabHistoryTableOrderingComposer,
      $$LabHistoryTableAnnotationComposer,
      $$LabHistoryTableCreateCompanionBuilder,
      $$LabHistoryTableUpdateCompanionBuilder,
      (
        LabHistoryData,
        BaseReferences<_$AppDatabase, $LabHistoryTable, LabHistoryData>,
      ),
      LabHistoryData,
      PrefetchHooks Function()
    >;
typedef $$NotificationsTableCreateCompanionBuilder =
    NotificationsCompanion Function({
      required String id,
      required String type,
      required String title,
      required String body,
      Value<String?> deepLink,
      required int timestamp,
      Value<bool> read,
      Value<int> rowid,
    });
typedef $$NotificationsTableUpdateCompanionBuilder =
    NotificationsCompanion Function({
      Value<String> id,
      Value<String> type,
      Value<String> title,
      Value<String> body,
      Value<String?> deepLink,
      Value<int> timestamp,
      Value<bool> read,
      Value<int> rowid,
    });

class $$NotificationsTableFilterComposer
    extends Composer<_$AppDatabase, $NotificationsTable> {
  $$NotificationsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get type => $composableBuilder(
    column: $table.type,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get title => $composableBuilder(
    column: $table.title,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get body => $composableBuilder(
    column: $table.body,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get deepLink => $composableBuilder(
    column: $table.deepLink,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<bool> get read => $composableBuilder(
    column: $table.read,
    builder: (column) => ColumnFilters(column),
  );
}

class $$NotificationsTableOrderingComposer
    extends Composer<_$AppDatabase, $NotificationsTable> {
  $$NotificationsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get type => $composableBuilder(
    column: $table.type,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get title => $composableBuilder(
    column: $table.title,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get body => $composableBuilder(
    column: $table.body,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get deepLink => $composableBuilder(
    column: $table.deepLink,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<bool> get read => $composableBuilder(
    column: $table.read,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$NotificationsTableAnnotationComposer
    extends Composer<_$AppDatabase, $NotificationsTable> {
  $$NotificationsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get type =>
      $composableBuilder(column: $table.type, builder: (column) => column);

  GeneratedColumn<String> get title =>
      $composableBuilder(column: $table.title, builder: (column) => column);

  GeneratedColumn<String> get body =>
      $composableBuilder(column: $table.body, builder: (column) => column);

  GeneratedColumn<String> get deepLink =>
      $composableBuilder(column: $table.deepLink, builder: (column) => column);

  GeneratedColumn<int> get timestamp =>
      $composableBuilder(column: $table.timestamp, builder: (column) => column);

  GeneratedColumn<bool> get read =>
      $composableBuilder(column: $table.read, builder: (column) => column);
}

class $$NotificationsTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $NotificationsTable,
          Notification,
          $$NotificationsTableFilterComposer,
          $$NotificationsTableOrderingComposer,
          $$NotificationsTableAnnotationComposer,
          $$NotificationsTableCreateCompanionBuilder,
          $$NotificationsTableUpdateCompanionBuilder,
          (
            Notification,
            BaseReferences<_$AppDatabase, $NotificationsTable, Notification>,
          ),
          Notification,
          PrefetchHooks Function()
        > {
  $$NotificationsTableTableManager(_$AppDatabase db, $NotificationsTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$NotificationsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$NotificationsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$NotificationsTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> type = const Value.absent(),
                Value<String> title = const Value.absent(),
                Value<String> body = const Value.absent(),
                Value<String?> deepLink = const Value.absent(),
                Value<int> timestamp = const Value.absent(),
                Value<bool> read = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => NotificationsCompanion(
                id: id,
                type: type,
                title: title,
                body: body,
                deepLink: deepLink,
                timestamp: timestamp,
                read: read,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String type,
                required String title,
                required String body,
                Value<String?> deepLink = const Value.absent(),
                required int timestamp,
                Value<bool> read = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => NotificationsCompanion.insert(
                id: id,
                type: type,
                title: title,
                body: body,
                deepLink: deepLink,
                timestamp: timestamp,
                read: read,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$NotificationsTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $NotificationsTable,
      Notification,
      $$NotificationsTableFilterComposer,
      $$NotificationsTableOrderingComposer,
      $$NotificationsTableAnnotationComposer,
      $$NotificationsTableCreateCompanionBuilder,
      $$NotificationsTableUpdateCompanionBuilder,
      (
        Notification,
        BaseReferences<_$AppDatabase, $NotificationsTable, Notification>,
      ),
      Notification,
      PrefetchHooks Function()
    >;
typedef $$ObservationQueueTableCreateCompanionBuilder =
    ObservationQueueCompanion Function({
      required String id,
      required String type,
      required String value,
      required String unit,
      required int timestamp,
      Value<bool> synced,
      Value<int> rowid,
    });
typedef $$ObservationQueueTableUpdateCompanionBuilder =
    ObservationQueueCompanion Function({
      Value<String> id,
      Value<String> type,
      Value<String> value,
      Value<String> unit,
      Value<int> timestamp,
      Value<bool> synced,
      Value<int> rowid,
    });

class $$ObservationQueueTableFilterComposer
    extends Composer<_$AppDatabase, $ObservationQueueTable> {
  $$ObservationQueueTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get type => $composableBuilder(
    column: $table.type,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get value => $composableBuilder(
    column: $table.value,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get unit => $composableBuilder(
    column: $table.unit,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<bool> get synced => $composableBuilder(
    column: $table.synced,
    builder: (column) => ColumnFilters(column),
  );
}

class $$ObservationQueueTableOrderingComposer
    extends Composer<_$AppDatabase, $ObservationQueueTable> {
  $$ObservationQueueTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get type => $composableBuilder(
    column: $table.type,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get value => $composableBuilder(
    column: $table.value,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get unit => $composableBuilder(
    column: $table.unit,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<bool> get synced => $composableBuilder(
    column: $table.synced,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$ObservationQueueTableAnnotationComposer
    extends Composer<_$AppDatabase, $ObservationQueueTable> {
  $$ObservationQueueTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get type =>
      $composableBuilder(column: $table.type, builder: (column) => column);

  GeneratedColumn<String> get value =>
      $composableBuilder(column: $table.value, builder: (column) => column);

  GeneratedColumn<String> get unit =>
      $composableBuilder(column: $table.unit, builder: (column) => column);

  GeneratedColumn<int> get timestamp =>
      $composableBuilder(column: $table.timestamp, builder: (column) => column);

  GeneratedColumn<bool> get synced =>
      $composableBuilder(column: $table.synced, builder: (column) => column);
}

class $$ObservationQueueTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $ObservationQueueTable,
          ObservationQueueData,
          $$ObservationQueueTableFilterComposer,
          $$ObservationQueueTableOrderingComposer,
          $$ObservationQueueTableAnnotationComposer,
          $$ObservationQueueTableCreateCompanionBuilder,
          $$ObservationQueueTableUpdateCompanionBuilder,
          (
            ObservationQueueData,
            BaseReferences<
              _$AppDatabase,
              $ObservationQueueTable,
              ObservationQueueData
            >,
          ),
          ObservationQueueData,
          PrefetchHooks Function()
        > {
  $$ObservationQueueTableTableManager(
    _$AppDatabase db,
    $ObservationQueueTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$ObservationQueueTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$ObservationQueueTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$ObservationQueueTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> type = const Value.absent(),
                Value<String> value = const Value.absent(),
                Value<String> unit = const Value.absent(),
                Value<int> timestamp = const Value.absent(),
                Value<bool> synced = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => ObservationQueueCompanion(
                id: id,
                type: type,
                value: value,
                unit: unit,
                timestamp: timestamp,
                synced: synced,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String type,
                required String value,
                required String unit,
                required int timestamp,
                Value<bool> synced = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => ObservationQueueCompanion.insert(
                id: id,
                type: type,
                value: value,
                unit: unit,
                timestamp: timestamp,
                synced: synced,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$ObservationQueueTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $ObservationQueueTable,
      ObservationQueueData,
      $$ObservationQueueTableFilterComposer,
      $$ObservationQueueTableOrderingComposer,
      $$ObservationQueueTableAnnotationComposer,
      $$ObservationQueueTableCreateCompanionBuilder,
      $$ObservationQueueTableUpdateCompanionBuilder,
      (
        ObservationQueueData,
        BaseReferences<
          _$AppDatabase,
          $ObservationQueueTable,
          ObservationQueueData
        >,
      ),
      ObservationQueueData,
      PrefetchHooks Function()
    >;
typedef $$MedicationLogTableCreateCompanionBuilder =
    MedicationLogCompanion Function({
      required String id,
      required String actionId,
      required String medicationName,
      required bool completed,
      required int timestamp,
      Value<int> rowid,
    });
typedef $$MedicationLogTableUpdateCompanionBuilder =
    MedicationLogCompanion Function({
      Value<String> id,
      Value<String> actionId,
      Value<String> medicationName,
      Value<bool> completed,
      Value<int> timestamp,
      Value<int> rowid,
    });

class $$MedicationLogTableFilterComposer
    extends Composer<_$AppDatabase, $MedicationLogTable> {
  $$MedicationLogTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get actionId => $composableBuilder(
    column: $table.actionId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get medicationName => $composableBuilder(
    column: $table.medicationName,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<bool> get completed => $composableBuilder(
    column: $table.completed,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnFilters(column),
  );
}

class $$MedicationLogTableOrderingComposer
    extends Composer<_$AppDatabase, $MedicationLogTable> {
  $$MedicationLogTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get actionId => $composableBuilder(
    column: $table.actionId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get medicationName => $composableBuilder(
    column: $table.medicationName,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<bool> get completed => $composableBuilder(
    column: $table.completed,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$MedicationLogTableAnnotationComposer
    extends Composer<_$AppDatabase, $MedicationLogTable> {
  $$MedicationLogTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get actionId =>
      $composableBuilder(column: $table.actionId, builder: (column) => column);

  GeneratedColumn<String> get medicationName => $composableBuilder(
    column: $table.medicationName,
    builder: (column) => column,
  );

  GeneratedColumn<bool> get completed =>
      $composableBuilder(column: $table.completed, builder: (column) => column);

  GeneratedColumn<int> get timestamp =>
      $composableBuilder(column: $table.timestamp, builder: (column) => column);
}

class $$MedicationLogTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $MedicationLogTable,
          MedicationLogData,
          $$MedicationLogTableFilterComposer,
          $$MedicationLogTableOrderingComposer,
          $$MedicationLogTableAnnotationComposer,
          $$MedicationLogTableCreateCompanionBuilder,
          $$MedicationLogTableUpdateCompanionBuilder,
          (
            MedicationLogData,
            BaseReferences<
              _$AppDatabase,
              $MedicationLogTable,
              MedicationLogData
            >,
          ),
          MedicationLogData,
          PrefetchHooks Function()
        > {
  $$MedicationLogTableTableManager(_$AppDatabase db, $MedicationLogTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$MedicationLogTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$MedicationLogTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$MedicationLogTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> actionId = const Value.absent(),
                Value<String> medicationName = const Value.absent(),
                Value<bool> completed = const Value.absent(),
                Value<int> timestamp = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => MedicationLogCompanion(
                id: id,
                actionId: actionId,
                medicationName: medicationName,
                completed: completed,
                timestamp: timestamp,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String actionId,
                required String medicationName,
                required bool completed,
                required int timestamp,
                Value<int> rowid = const Value.absent(),
              }) => MedicationLogCompanion.insert(
                id: id,
                actionId: actionId,
                medicationName: medicationName,
                completed: completed,
                timestamp: timestamp,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$MedicationLogTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $MedicationLogTable,
      MedicationLogData,
      $$MedicationLogTableFilterComposer,
      $$MedicationLogTableOrderingComposer,
      $$MedicationLogTableAnnotationComposer,
      $$MedicationLogTableCreateCompanionBuilder,
      $$MedicationLogTableUpdateCompanionBuilder,
      (
        MedicationLogData,
        BaseReferences<_$AppDatabase, $MedicationLogTable, MedicationLogData>,
      ),
      MedicationLogData,
      PrefetchHooks Function()
    >;
typedef $$SymptomLogTableCreateCompanionBuilder =
    SymptomLogCompanion Function({
      required String id,
      required String symptom,
      required String severity,
      Value<String?> notes,
      required int timestamp,
      Value<bool> synced,
      Value<int> rowid,
    });
typedef $$SymptomLogTableUpdateCompanionBuilder =
    SymptomLogCompanion Function({
      Value<String> id,
      Value<String> symptom,
      Value<String> severity,
      Value<String?> notes,
      Value<int> timestamp,
      Value<bool> synced,
      Value<int> rowid,
    });

class $$SymptomLogTableFilterComposer
    extends Composer<_$AppDatabase, $SymptomLogTable> {
  $$SymptomLogTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get symptom => $composableBuilder(
    column: $table.symptom,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get severity => $composableBuilder(
    column: $table.severity,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get notes => $composableBuilder(
    column: $table.notes,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<bool> get synced => $composableBuilder(
    column: $table.synced,
    builder: (column) => ColumnFilters(column),
  );
}

class $$SymptomLogTableOrderingComposer
    extends Composer<_$AppDatabase, $SymptomLogTable> {
  $$SymptomLogTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get symptom => $composableBuilder(
    column: $table.symptom,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get severity => $composableBuilder(
    column: $table.severity,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get notes => $composableBuilder(
    column: $table.notes,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<bool> get synced => $composableBuilder(
    column: $table.synced,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$SymptomLogTableAnnotationComposer
    extends Composer<_$AppDatabase, $SymptomLogTable> {
  $$SymptomLogTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get symptom =>
      $composableBuilder(column: $table.symptom, builder: (column) => column);

  GeneratedColumn<String> get severity =>
      $composableBuilder(column: $table.severity, builder: (column) => column);

  GeneratedColumn<String> get notes =>
      $composableBuilder(column: $table.notes, builder: (column) => column);

  GeneratedColumn<int> get timestamp =>
      $composableBuilder(column: $table.timestamp, builder: (column) => column);

  GeneratedColumn<bool> get synced =>
      $composableBuilder(column: $table.synced, builder: (column) => column);
}

class $$SymptomLogTableTableManager
    extends
        RootTableManager<
          _$AppDatabase,
          $SymptomLogTable,
          SymptomLogData,
          $$SymptomLogTableFilterComposer,
          $$SymptomLogTableOrderingComposer,
          $$SymptomLogTableAnnotationComposer,
          $$SymptomLogTableCreateCompanionBuilder,
          $$SymptomLogTableUpdateCompanionBuilder,
          (
            SymptomLogData,
            BaseReferences<_$AppDatabase, $SymptomLogTable, SymptomLogData>,
          ),
          SymptomLogData,
          PrefetchHooks Function()
        > {
  $$SymptomLogTableTableManager(_$AppDatabase db, $SymptomLogTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$SymptomLogTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$SymptomLogTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$SymptomLogTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> symptom = const Value.absent(),
                Value<String> severity = const Value.absent(),
                Value<String?> notes = const Value.absent(),
                Value<int> timestamp = const Value.absent(),
                Value<bool> synced = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => SymptomLogCompanion(
                id: id,
                symptom: symptom,
                severity: severity,
                notes: notes,
                timestamp: timestamp,
                synced: synced,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String symptom,
                required String severity,
                Value<String?> notes = const Value.absent(),
                required int timestamp,
                Value<bool> synced = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => SymptomLogCompanion.insert(
                id: id,
                symptom: symptom,
                severity: severity,
                notes: notes,
                timestamp: timestamp,
                synced: synced,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$SymptomLogTableProcessedTableManager =
    ProcessedTableManager<
      _$AppDatabase,
      $SymptomLogTable,
      SymptomLogData,
      $$SymptomLogTableFilterComposer,
      $$SymptomLogTableOrderingComposer,
      $$SymptomLogTableAnnotationComposer,
      $$SymptomLogTableCreateCompanionBuilder,
      $$SymptomLogTableUpdateCompanionBuilder,
      (
        SymptomLogData,
        BaseReferences<_$AppDatabase, $SymptomLogTable, SymptomLogData>,
      ),
      SymptomLogData,
      PrefetchHooks Function()
    >;

class $AppDatabaseManager {
  final _$AppDatabase _db;
  $AppDatabaseManager(this._db);
  $$CheckinQueueTableTableManager get checkinQueue =>
      $$CheckinQueueTableTableManager(_db, _db.checkinQueue);
  $$LabHistoryTableTableManager get labHistory =>
      $$LabHistoryTableTableManager(_db, _db.labHistory);
  $$NotificationsTableTableManager get notifications =>
      $$NotificationsTableTableManager(_db, _db.notifications);
  $$ObservationQueueTableTableManager get observationQueue =>
      $$ObservationQueueTableTableManager(_db, _db.observationQueue);
  $$MedicationLogTableTableManager get medicationLog =>
      $$MedicationLogTableTableManager(_db, _db.medicationLog);
  $$SymptomLogTableTableManager get symptomLog =>
      $$SymptomLogTableTableManager(_db, _db.symptomLog);
}
