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

@DriftDatabase(tables: [CheckinQueue, LabHistory])
class AppDatabase extends _$AppDatabase {
  AppDatabase(QueryExecutor e) : super(e);

  @override
  int get schemaVersion => 1;

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
