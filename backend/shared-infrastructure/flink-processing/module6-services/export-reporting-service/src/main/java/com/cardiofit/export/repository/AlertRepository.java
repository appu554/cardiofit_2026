package com.cardiofit.export.repository;

import com.cardiofit.export.model.Alert;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;

/**
 * Repository for alert data access
 */
@Repository
public interface AlertRepository extends JpaRepository<Alert, String> {

    @Query("SELECT a FROM Alert a " +
           "WHERE a.department = :department " +
           "AND a.createdAt BETWEEN :startTime AND :endTime " +
           "ORDER BY a.createdAt DESC")
    List<Alert> findByDepartmentAndTimeRange(
            @Param("department") String department,
            @Param("startTime") Long startTime,
            @Param("endTime") Long endTime
    );

    @Query("SELECT a FROM Alert a " +
           "WHERE a.createdAt BETWEEN :startTime AND :endTime " +
           "ORDER BY a.createdAt DESC")
    List<Alert> findAllByTimeRange(
            @Param("startTime") Long startTime,
            @Param("endTime") Long endTime
    );
}
