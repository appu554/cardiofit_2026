import analyticsDataService from '../services/analytics-data.service';
import { PatientRiskProfile, RiskLevel } from '../models/types';
import { logger } from '../config';

export const patientRiskResolvers = {
  Query: {
    patientRiskProfile: async (
      _: any,
      { patientId }: { patientId: string }
    ): Promise<PatientRiskProfile | null> => {
      try {
        return await analyticsDataService.getPatientRiskProfile(patientId);
      } catch (error) {
        logger.error({ error, patientId }, 'Error fetching patient risk profile');
        throw new Error('Failed to fetch patient risk profile');
      }
    },

    highRiskPatients: async (
      _: any,
      {
        hospitalId,
        riskLevel,
        limit,
      }: { hospitalId: string; riskLevel?: RiskLevel; limit?: number }
    ): Promise<PatientRiskProfile[]> => {
      try {
        return await analyticsDataService.getHighRiskPatients(hospitalId, riskLevel, limit);
      } catch (error) {
        logger.error({ error, hospitalId }, 'Error fetching high risk patients');
        throw new Error('Failed to fetch high risk patients');
      }
    },

    patientRiskHistory: async (
      _: any,
      {
        patientId,
        startTime,
        endTime,
      }: { patientId: string; startTime: Date; endTime: Date }
    ): Promise<PatientRiskProfile[]> => {
      try {
        // Implementation would query time-series data
        return [];
      } catch (error) {
        logger.error({ error, patientId }, 'Error fetching patient risk history');
        throw new Error('Failed to fetch patient risk history');
      }
    },
  },

  PatientRiskProfile: {
    timestamp: (parent: PatientRiskProfile) => parent.timestamp.toISOString(),
    lastUpdated: (parent: PatientRiskProfile) => parent.lastUpdated.toISOString(),
    vitalSigns: (parent: PatientRiskProfile) => {
      if (!parent.vitalSigns) return null;
      return {
        ...parent.vitalSigns,
        timestamp: parent.vitalSigns.timestamp.toISOString(),
      };
    },
    labResults: (parent: PatientRiskProfile) => {
      if (!parent.labResults) return null;
      return {
        ...parent.labResults,
        timestamp: parent.labResults.timestamp.toISOString(),
      };
    },
    activeAlerts: (parent: PatientRiskProfile) => {
      if (!parent.activeAlerts) return [];
      return parent.activeAlerts.map((alert) => ({
        ...alert,
        timestamp: alert.timestamp.toISOString(),
      }));
    },
  },
};
