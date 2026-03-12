import analyticsDataService from '../services/analytics-data.service';
import { SepsisSurveillance, AlertLevel } from '../models/types';
import { logger } from '../config';

export const sepsisSurveillanceResolvers = {
  Query: {
    sepsisSurveillance: async (
      _: any,
      { hospitalId, alertLevel }: { hospitalId: string; alertLevel?: AlertLevel }
    ): Promise<SepsisSurveillance[]> => {
      try {
        return await analyticsDataService.getSepsisSurveillance(hospitalId, alertLevel);
      } catch (error) {
        logger.error({ error, hospitalId }, 'Error fetching sepsis surveillance');
        throw new Error('Failed to fetch sepsis surveillance');
      }
    },

    sepsisAlerts: async (
      _: any,
      {
        hospitalId,
        startTime,
        limit,
      }: { hospitalId: string; startTime?: Date; limit?: number }
    ): Promise<SepsisSurveillance[]> => {
      try {
        return await analyticsDataService.getSepsisSurveillance(hospitalId);
      } catch (error) {
        logger.error({ error, hospitalId }, 'Error fetching sepsis alerts');
        throw new Error('Failed to fetch sepsis alerts');
      }
    },

    sepsisPatientDetails: async (
      _: any,
      { patientId }: { patientId: string }
    ): Promise<SepsisSurveillance | null> => {
      try {
        return await analyticsDataService.getSepsisPatientDetails(patientId);
      } catch (error) {
        logger.error({ error, patientId }, 'Error fetching sepsis patient details');
        throw new Error('Failed to fetch sepsis patient details');
      }
    },
  },

  SepsisSurveillance: {
    timestamp: (parent: SepsisSurveillance) => parent.timestamp.toISOString(),
    alertGeneratedTime: (parent: SepsisSurveillance) =>
      parent.alertGeneratedTime.toISOString(),
    symptomOnsetTime: (parent: SepsisSurveillance) =>
      parent.symptomOnsetTime ? parent.symptomOnsetTime.toISOString() : null,
    clinicianNotifiedTime: (parent: SepsisSurveillance) =>
      parent.clinicianNotifiedTime ? parent.clinicianNotifiedTime.toISOString() : null,
    interventionStartTime: (parent: SepsisSurveillance) =>
      parent.interventionStartTime ? parent.interventionStartTime.toISOString() : null,
    resolutionTime: (parent: SepsisSurveillance) =>
      parent.resolutionTime ? parent.resolutionTime.toISOString() : null,
    antibioticStartTime: (parent: SepsisSurveillance) =>
      parent.antibioticStartTime ? parent.antibioticStartTime.toISOString() : null,
    vitalSigns: (parent: SepsisSurveillance) => ({
      ...parent.vitalSigns,
      timestamp: parent.vitalSigns.timestamp.toISOString(),
    }),
    labResults: (parent: SepsisSurveillance) => ({
      ...parent.labResults,
      timestamp: parent.labResults.timestamp.toISOString(),
    }),
  },
};
