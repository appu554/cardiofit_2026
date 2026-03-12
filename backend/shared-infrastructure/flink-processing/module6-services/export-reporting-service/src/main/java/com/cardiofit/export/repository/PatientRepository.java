package com.cardiofit.export.repository;

import com.cardiofit.export.model.PatientCurrentState;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;

/**
 * Repository for patient data access
 */
@Repository
public interface PatientRepository extends JpaRepository<PatientCurrentState, String> {

    @Query("SELECT p FROM PatientCurrentState p " +
           "WHERE p.departmentId = :departmentId " +
           "AND p.admissionTime BETWEEN :startTime AND :endTime " +
           "ORDER BY p.overallRiskScore DESC")
    List<PatientCurrentState> findByDepartmentAndTimeRange(
            @Param("departmentId") String departmentId,
            @Param("startTime") Long startTime,
            @Param("endTime") Long endTime
    );

    @Query("SELECT p FROM PatientCurrentState p " +
           "WHERE p.admissionTime BETWEEN :startTime AND :endTime " +
           "ORDER BY p.overallRiskScore DESC")
    List<PatientCurrentState> findAllByTimeRange(
            @Param("startTime") Long startTime,
            @Param("endTime") Long endTime
    );
}
