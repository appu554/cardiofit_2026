package com.cardiofit.export.repository;

import com.cardiofit.export.model.MlPrediction;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;

/**
 * Repository for ML prediction data access
 */
@Repository
public interface MlPredictionRepository extends JpaRepository<MlPrediction, String> {

    @Query("SELECT m FROM MlPrediction m " +
           "WHERE m.department = :department " +
           "AND m.modelType = :modelType " +
           "AND m.predictionTimestamp BETWEEN :startTime AND :endTime " +
           "ORDER BY m.predictionTimestamp DESC")
    List<MlPrediction> findByDepartmentModelTypeAndTimeRange(
            @Param("department") String department,
            @Param("modelType") String modelType,
            @Param("startTime") Long startTime,
            @Param("endTime") Long endTime
    );
}
