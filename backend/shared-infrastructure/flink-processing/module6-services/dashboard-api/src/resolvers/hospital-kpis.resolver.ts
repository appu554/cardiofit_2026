import analyticsDataService from '../services/analytics-data.service';
import { HospitalKPIs, TimeRange } from '../models/types';
import { logger } from '../config';

export const hospitalKpisResolvers = {
  Query: {
    hospitalKpis: async (
      _: any,
      { hospitalId, timeRange }: { hospitalId: string; timeRange?: TimeRange }
    ): Promise<HospitalKPIs | null> => {
      try {
        return await analyticsDataService.getHospitalKpis(hospitalId);
      } catch (error) {
        logger.error({ error, hospitalId }, 'Error fetching hospital KPIs');
        throw new Error('Failed to fetch hospital KPIs');
      }
    },

    hospitalKpisTrend: async (
      _: any,
      {
        hospitalId,
        startTime,
        endTime,
      }: { hospitalId: string; startTime: Date; endTime: Date }
    ): Promise<HospitalKPIs[]> => {
      try {
        return await analyticsDataService.getHospitalKpisTrend(hospitalId, startTime, endTime);
      } catch (error) {
        logger.error({ error, hospitalId }, 'Error fetching hospital KPIs trend');
        throw new Error('Failed to fetch hospital KPIs trend');
      }
    },
  },

  HospitalKPIs: {
    timestamp: (parent: HospitalKPIs) =>
      typeof parent.timestamp === 'string' ? parent.timestamp : parent.timestamp.toISOString(),
    windowStart: (parent: HospitalKPIs) =>
      typeof parent.windowStart === 'string' ? parent.windowStart : parent.windowStart.toISOString(),
    windowEnd: (parent: HospitalKPIs) =>
      typeof parent.windowEnd === 'string' ? parent.windowEnd : parent.windowEnd.toISOString(),
  },
};
