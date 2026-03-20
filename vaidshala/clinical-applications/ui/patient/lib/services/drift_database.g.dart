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

abstract class _$AppDatabase extends GeneratedDatabase {
  _$AppDatabase(QueryExecutor e) : super(e);
  $AppDatabaseManager get managers => $AppDatabaseManager(this);
  late final $CheckinQueueTable checkinQueue = $CheckinQueueTable(this);
  late final $LabHistoryTable labHistory = $LabHistoryTable(this);
  @override
  Iterable<TableInfo<Table, Object?>> get allTables =>
      allSchemaEntities.whereType<TableInfo<Table, Object?>>();
  @override
  List<DatabaseSchemaEntity> get allSchemaEntities => [
    checkinQueue,
    labHistory,
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

class $AppDatabaseManager {
  final _$AppDatabase _db;
  $AppDatabaseManager(this._db);
  $$CheckinQueueTableTableManager get checkinQueue =>
      $$CheckinQueueTableTableManager(_db, _db.checkinQueue);
  $$LabHistoryTableTableManager get labHistory =>
      $$LabHistoryTableTableManager(_db, _db.labHistory);
}
