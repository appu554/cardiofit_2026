import analyticsDataService from '../services/analytics-data.service';
import cacheService from '../services/cache.service';
import { logger } from '../config';

export const dashboardResolvers = {
  Query: {
    dashboardSummary: async (_: any, { hospitalId }: { hospitalId: string }) => {
      try {
        const [hospitalKpis, departmentMetrics, highRiskPatients, activeSepsisAlerts] =
          await Promise.all([
            analyticsDataService.getHospitalKpis(hospitalId),
            analyticsDataService.getDepartmentMetrics(hospitalId),
            analyticsDataService.getHighRiskPatients(hospitalId, undefined, 10),
            analyticsDataService.getSepsisSurveillance(hospitalId),
          ]);

        const realtimeStats = {
          hospitalId,
          timestamp: new Date().toISOString(),
          activePatients: hospitalKpis?.occupiedBeds || 0,
          criticalPatients: highRiskPatients.filter((p) => p.riskLevel === 'CRITICAL').length,
          activeSepsisAlerts: activeSepsisAlerts.length,
          pendingDischarges: 0, // Would be calculated from actual data
          availableBeds: hospitalKpis?.availableBeds || 0,
          staffOnDuty: 0, // Would be calculated from actual data
          lastUpdated: new Date().toISOString(),
        };

        return {
          hospitalId,
          timestamp: new Date().toISOString(),
          hospitalKpis,
          topDepartments: departmentMetrics.slice(0, 5),
          highRiskPatients,
          activeSepsisAlerts: activeSepsisAlerts.slice(0, 10),
          qualitySummary: {},
          realtimeStats,
        };
      } catch (error) {
        logger.error({ error, hospitalId }, 'Error fetching dashboard summary');
        throw new Error('Failed to fetch dashboard summary');
      }
    },

    realtimeStats: async (_: any, { hospitalId }: { hospitalId: string }) => {
      try {
        const cacheKey = `realtime-stats:${hospitalId}`;
        const cached = await cacheService.get(cacheKey);
        if (cached) return cached;

        const [hospitalKpis, highRiskPatients, activeSepsisAlerts] = await Promise.all([
          analyticsDataService.getHospitalKpis(hospitalId),
          analyticsDataService.getHighRiskPatients(hospitalId, undefined, 1000),
          analyticsDataService.getSepsisSurveillance(hospitalId),
        ]);

        const stats = {
          hospitalId,
          timestamp: new Date().toISOString(),
          activePatients: hospitalKpis?.occupiedBeds || 0,
          criticalPatients: highRiskPatients.filter((p) => p.riskLevel === 'CRITICAL').length,
          activeSepsisAlerts: activeSepsisAlerts.length,
          pendingDischarges: 0,
          availableBeds: hospitalKpis?.availableBeds || 0,
          staffOnDuty: 0,
          lastUpdated: new Date().toISOString(),
        };

        await cacheService.set(cacheKey, stats, 30); // Cache for 30 seconds
        return stats;
      } catch (error) {
        logger.error({ error, hospitalId }, 'Error fetching realtime stats');
        throw new Error('Failed to fetch realtime stats');
      }
    },
  },
};
