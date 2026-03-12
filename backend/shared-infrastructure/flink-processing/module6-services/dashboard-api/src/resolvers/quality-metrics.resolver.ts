import analyticsDataService from '../services/analytics-data.service';
import { QualityMetrics, QualityMetricType, TimeRange, TrendDirection } from '../models/types';
import { logger } from '../config';

export const qualityMetricsResolvers = {
  Query: {
    qualityMetrics: async (
      _: any,
      {
        hospitalId,
        metricType,
        timeRange,
      }: { hospitalId: string; metricType?: QualityMetricType; timeRange?: TimeRange }
    ): Promise<QualityMetrics[]> => {
      try {
        return await analyticsDataService.getQualityMetrics(hospitalId, metricType);
      } catch (error) {
        logger.error({ error, hospitalId, metricType }, 'Error fetching quality metrics');
        throw new Error('Failed to fetch quality metrics');
      }
    },

    qualityMetricsTrend: async (
      _: any,
      {
        hospitalId,
        metricType,
        startTime,
        endTime,
      }: {
        hospitalId: string;
        metricType: QualityMetricType;
        startTime: Date;
        endTime: Date;
      }
    ): Promise<QualityMetrics[]> => {
      try {
        // Implementation would query time-series data
        return [];
      } catch (error) {
        logger.error({ error, hospitalId, metricType }, 'Error fetching quality metrics trend');
        throw new Error('Failed to fetch quality metrics trend');
      }
    },

    complianceScore: async (
      _: any,
      { hospitalId, timeRange }: { hospitalId: string; timeRange?: TimeRange }
    ): Promise<{
      hospitalId: string;
      timestamp: string;
      overallScore: number;
      categoryScores: Record<string, any>;
      trendDirection: TrendDirection;
    }> => {
      try {
        const metrics = await analyticsDataService.getQualityMetrics(hospitalId);

        // Calculate overall compliance score
        const complianceMetrics = metrics.filter((m) => m.complianceRate !== undefined);
        const overallScore =
          complianceMetrics.length > 0
            ? complianceMetrics.reduce((sum, m) => sum + (m.complianceRate || 0), 0) /
              complianceMetrics.length
            : 0;

        // Group by category
        const categoryScores: Record<string, any> = {};
        metrics.forEach((metric) => {
          categoryScores[metric.metricType] = metric.complianceRate || metric.metricValue;
        });

        return {
          hospitalId,
          timestamp: new Date().toISOString(),
          overallScore,
          categoryScores,
          trendDirection: TrendDirection.STABLE,
        };
      } catch (error) {
        logger.error({ error, hospitalId }, 'Error calculating compliance score');
        throw new Error('Failed to calculate compliance score');
      }
    },

    // Quality Metrics Dashboard Queries
    bundleCompliance: async (
      _: any,
      { departmentId, period }: { departmentId?: string; period: string }
    ): Promise<any[]> => {
      try {
        return await analyticsDataService.getBundleCompliance(departmentId, period);
      } catch (error) {
        logger.error({ error, departmentId, period }, 'Error fetching bundle compliance');
        throw new Error('Failed to fetch bundle compliance');
      }
    },

    outcomeMetrics: async (
      _: any,
      { departmentId }: { departmentId?: string }
    ): Promise<any[]> => {
      try {
        return await analyticsDataService.getOutcomeMetrics(departmentId);
      } catch (error) {
        logger.error({ error, departmentId }, 'Error fetching outcome metrics');
        throw new Error('Failed to fetch outcome metrics');
      }
    },

    departmentQualityComparison: async (): Promise<any[]> => {
      try {
        return await analyticsDataService.getDepartmentQualityComparison();
      } catch (error) {
        logger.error({ error }, 'Error fetching department quality comparison');
        throw new Error('Failed to fetch department quality comparison');
      }
    },
  },

  QualityMetrics: {
    timestamp: (parent: QualityMetrics) => parent.timestamp.toISOString(),
    windowStart: (parent: QualityMetrics) => parent.windowStart.toISOString(),
    windowEnd: (parent: QualityMetrics) => parent.windowEnd.toISOString(),
  },
};
