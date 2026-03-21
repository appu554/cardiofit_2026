import 'package:drift/drift.dart';
import 'package:drift/wasm.dart';

part 'drift_database.g.dart';

class CheckinQueue extends Table {
  IntColumn get id => integer().autoIncrement()();
  TextColumn get actionId => text()();
  BoolColumn get completed => boolean()();
  DateTimeColumn get timestamp => dateTime()();
  BoolColumn get synced => boolean().withDefault(const Constant(false))();
}

class LabHistory extends Table {
  IntColumn get id => integer().autoIncrement()();
  TextColumn get metricId => text()();
  RealColumn get value => real()();
  TextColumn get unit => text()();
  DateTimeColumn get recordedAt => dateTime()();
}

class Notifications extends Table {
  TextColumn get id => text()();
  TextColumn get type => text()();          // coaching, reminder, alert, milestone
  TextColumn get title => text()();
  TextColumn get body => text()();
  TextColumn get deepLink => text().nullable()();
  IntColumn get timestamp => integer()();   // epoch ms
  BoolColumn get read => boolean().withDefault(const Constant(false))();

  @override
  Set<Column> get primaryKey => {id};
}

class ObservationQueue extends Table {
  TextColumn get id => text()();
  TextColumn get type => text()();          // bp, glucose, weight
  TextColumn get value => text()();         // JSON string
  TextColumn get unit => text()();
  IntColumn get timestamp => integer()();   // epoch ms
  BoolColumn get synced => boolean().withDefault(const Constant(false))();

  @override
  Set<Column> get primaryKey => {id};
}

class MedicationLog extends Table {
  TextColumn get id => text()();
  TextColumn get actionId => text()();
  TextColumn get medicationName => text()();
  BoolColumn get completed => boolean()();  // true = taken, false = missed
  IntColumn get timestamp => integer()();   // epoch ms

  @override
  Set<Column> get primaryKey => {id};
}

class SymptomLog extends Table {
  TextColumn get id => text()();
  TextColumn get symptom => text()();       // comma-separated
  TextColumn get severity => text()();      // mild, moderate, severe
  TextColumn get notes => text().nullable()();
  IntColumn get timestamp => integer()();   // epoch ms
  BoolColumn get synced => boolean().withDefault(const Constant(false))();

  @override
  Set<Column> get primaryKey => {id};
}

@DriftDatabase(tables: [CheckinQueue, LabHistory, Notifications, ObservationQueue, MedicationLog, SymptomLog])
class AppDatabase extends _$AppDatabase {
  AppDatabase(super.e);

  @override
  int get schemaVersion => 2;

  @override
  MigrationStrategy get migration => MigrationStrategy(
        onCreate: (m) => m.createAll(),
        onUpgrade: (m, from, to) async {
          if (from < 2) {
            await m.createTable(notifications);
            await m.createTable(observationQueue);
            await m.createTable(medicationLog);
            await m.createTable(symptomLog);
          }
        },
      );

  // Checkin queue operations
  Future<int> queueCheckin({
    required String actionId,
    required bool completed,
  }) =>
      into(checkinQueue).insert(CheckinQueueCompanion.insert(
        actionId: actionId,
        completed: completed,
        timestamp: DateTime.now(),
      ));

  Future<List<CheckinQueueData>> pendingCheckins() =>
      (select(checkinQueue)..where((t) => t.synced.equals(false))).get();

  Future<void> markSynced(int id) =>
      (update(checkinQueue)..where((t) => t.id.equals(id)))
          .write(const CheckinQueueCompanion(synced: Value(true)));

  // Lab history operations
  Future<int> insertLab({
    required String metricId,
    required double value,
    required String unit,
  }) =>
      into(labHistory).insert(LabHistoryCompanion.insert(
        metricId: metricId,
        value: value,
        unit: unit,
        recordedAt: DateTime.now(),
      ));

  Future<List<LabHistoryData>> labsForMetric(String metricId) =>
      (select(labHistory)
            ..where((t) => t.metricId.equals(metricId))
            ..orderBy([(t) => OrderingTerm.asc(t.recordedAt)]))
          .get();

  // Notification operations
  Future<void> seedNotifications(List<NotificationsCompanion> items) async {
    final count = await (selectOnly(notifications)..addColumns([notifications.id.count()])).getSingle();
    if ((count.read(notifications.id.count()) ?? 0) > 0) return;
    await batch((b) => b.insertAll(notifications, items));
  }

  Future<List<Notification>> allNotifications() =>
      (select(notifications)..orderBy([(t) => OrderingTerm.desc(t.timestamp)])).get();

  Future<void> markNotificationRead(String id) =>
      (update(notifications)..where((t) => t.id.equals(id)))
          .write(const NotificationsCompanion(read: Value(true)));

  Future<void> markAllNotificationsRead() =>
      update(notifications).write(const NotificationsCompanion(read: Value(true)));

  Future<void> deleteNotification(String id) =>
      (delete(notifications)..where((t) => t.id.equals(id))).go();

  Future<int> unreadNotificationCount() async {
    final query = selectOnly(notifications)
      ..addColumns([notifications.id.count()])
      ..where(notifications.read.equals(false));
    final result = await query.getSingle();
    return result.read(notifications.id.count()) ?? 0;
  }

  // Observation queue operations
  Future<void> insertObservation({
    required String id,
    required String type,
    required String value,
    required String unit,
  }) =>
      into(observationQueue).insert(ObservationQueueCompanion.insert(
        id: id,
        type: type,
        value: value,
        unit: unit,
        timestamp: DateTime.now().millisecondsSinceEpoch,
      ));

  Future<List<ObservationQueueData>> recentObservations(String type) =>
      (select(observationQueue)
            ..where((t) => t.type.equals(type))
            ..orderBy([(t) => OrderingTerm.desc(t.timestamp)])
            ..limit(20))
          .get();

  // Medication log operations
  Future<void> logMedication({
    required String id,
    required String actionId,
    required String medicationName,
    required bool completed,
  }) =>
      into(medicationLog).insert(MedicationLogCompanion.insert(
        id: id,
        actionId: actionId,
        medicationName: medicationName,
        completed: completed,
        timestamp: DateTime.now().millisecondsSinceEpoch,
      ));

  Future<List<MedicationLogData>> medicationHistory({int days = 14}) {
    final cutoff = DateTime.now().subtract(Duration(days: days)).millisecondsSinceEpoch;
    return (select(medicationLog)
          ..where((t) => t.timestamp.isBiggerOrEqualValue(cutoff))
          ..orderBy([(t) => OrderingTerm.desc(t.timestamp)]))
        .get();
  }

  // Symptom log operations
  Future<void> insertSymptom({
    required String id,
    required String symptom,
    required String severity,
    String? notes,
  }) =>
      into(symptomLog).insert(SymptomLogCompanion.insert(
        id: id,
        symptom: symptom,
        severity: severity,
        notes: Value(notes),
        timestamp: DateTime.now().millisecondsSinceEpoch,
      ));

  Future<List<SymptomLogData>> recentSymptoms({int days = 30}) {
    final cutoff = DateTime.now().subtract(Duration(days: days)).millisecondsSinceEpoch;
    return (select(symptomLog)
          ..where((t) => t.timestamp.isBiggerOrEqualValue(cutoff))
          ..orderBy([(t) => OrderingTerm.desc(t.timestamp)]))
        .get();
  }
}

AppDatabase constructDb() {
  final db = LazyDatabase(() async {
    final result = await WasmDatabase.open(
      databaseName: 'patient_db',
      sqlite3Uri: Uri.parse('sqlite3.wasm'),
      driftWorkerUri: Uri.parse('drift_worker.dart.js'),
    );
    return result.resolvedExecutor;
  });
  return AppDatabase(db);
}
