import analyticsDataService from '../services/analytics-data.service';
import { DepartmentMetrics, TimeRange } from '../models/types';
import { logger } from '../config';

export const departmentMetricsResolvers = {
  Query: {
    departmentMetrics: async (
      _: any,
      {
        hospitalId,
        departmentId,
        timeRange,
      }: { hospitalId: string; departmentId?: string; timeRange?: TimeRange }
    ): Promise<DepartmentMetrics[]> => {
      try {
        return await analyticsDataService.getDepartmentMetrics(hospitalId, departmentId);
      } catch (error) {
        logger.error({ error, hospitalId, departmentId }, 'Error fetching department metrics');
        throw new Error('Failed to fetch department metrics');
      }
    },

    departmentMetricsTrend: async (
      _: any,
      {
        departmentId,
        startTime,
        endTime,
      }: { departmentId: string; startTime: Date; endTime: Date }
    ): Promise<DepartmentMetrics[]> => {
      try {
        // Implementation would query time-series data
        return [];
      } catch (error) {
        logger.error({ error, departmentId }, 'Error fetching department metrics trend');
        throw new Error('Failed to fetch department metrics trend');
      }
    },
  },

  DepartmentMetrics: {
    timestamp: (parent: DepartmentMetrics) => {
      if (parent.timestamp instanceof Date) {
        return parent.timestamp.toISOString();
      }
      if (typeof parent.timestamp === 'string') {
        return parent.timestamp;
      }
      if (typeof parent.timestamp === 'number') {
        return new Date(parent.timestamp).toISOString();
      }
      return new Date().toISOString();
    },
  },
};
