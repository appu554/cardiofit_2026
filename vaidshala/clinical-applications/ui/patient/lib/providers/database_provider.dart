import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../services/drift_database.dart';

final databaseProvider = Provider<AppDatabase>((ref) {
  final db = constructDb();
  ref.onDispose(() => db.close());
  return db;
});
