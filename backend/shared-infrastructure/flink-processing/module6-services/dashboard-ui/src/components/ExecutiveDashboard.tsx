import React, { useState } from 'react';
import {
  Grid,
  Paper,
  Typography,
  Box,
  Alert,
  FormControl,
  Select,
  MenuItem,
  SelectChangeEvent,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  People,
  Warning,
  LocalHospital,
  TrendingUp,
  Hotel,
} from '@mui/icons-material';
import { useQuery } from '@apollo/client';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import MetricCard from './MetricCard';
import {
  GET_HOSPITAL_METRICS,
  GET_RISK_TRENDS,
  HospitalMetrics,
  RiskTrend,
} from '../graphql/queries-fixed';

const RISK_COLORS = {
  high: '#d32f2f',
  medium: '#ed6c02',
  low: '#2e7d32',
};

const ExecutiveDashboard: React.FC = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [timeRange, setTimeRange] = useState('24');

  const { data: metricsData, loading: metricsLoading, error: metricsError } = useQuery<{
    dashboardSummary: {
      hospitalKpis: HospitalMetrics;
      realtimeStats: { activePatients: number; criticalPatients: number; activeSepsisAlerts: number; availableBeds: number; };
    };
  }>(GET_HOSPITAL_METRICS, {
    variables: { hospitalId: 'HOSPITAL-001' },
    pollInterval: 30000,
  });

  const { data: trendsData, loading: trendsLoading } = useQuery<{
    departmentMetrics: RiskTrend[];
  }>(GET_RISK_TRENDS, {
    variables: { hospitalId: 'HOSPITAL-001' },
    pollInterval: 30000,
  });

  const handleTimeRangeChange = (event: SelectChangeEvent) => {
    setTimeRange(event.target.value);
  };

  if (metricsError) {
    return (
      <Alert severity="error" sx={{ mt: 2 }}>
        Failed to load hospital metrics: {metricsError.message}
      </Alert>
    );
  }

  const summary = metricsData?.dashboardSummary;
  const metrics = summary?.hospitalKpis;
  const realtimeStats = summary?.realtimeStats;
  const departments = trendsData?.departmentMetrics || [];

  // Prepare data for charts using actual API fields
  const riskDistributionData = realtimeStats
    ? [
        { name: 'High Risk', value: realtimeStats.criticalPatients, color: RISK_COLORS.high },
        { name: 'Active', value: Math.max(0, realtimeStats.activePatients - realtimeStats.criticalPatients), color: RISK_COLORS.medium },
      ]
    : [];

  const departmentData = departments.map((dept) => ({
    name: dept.departmentId,
    patients: dept.totalPatients,
    highRisk: dept.highRiskPatients,
    alerts: dept.criticalPatients,
    avgRisk: dept.avgMortalityRisk,
  }));

  const trendChartData = departments.map((dept) => ({
    time: new Date(dept.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    High: dept.highRiskPatients,
    Critical: dept.criticalPatients,
    Total: dept.totalPatients,
  }));

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" fontWeight={600}>
          Executive Dashboard
        </Typography>
        <FormControl size="small" sx={{ minWidth: 120 }}>
          <Select value={timeRange} onChange={handleTimeRangeChange}>
            <MenuItem value="6">Last 6 Hours</MenuItem>
            <MenuItem value="12">Last 12 Hours</MenuItem>
            <MenuItem value="24">Last 24 Hours</MenuItem>
            <MenuItem value="48">Last 48 Hours</MenuItem>
          </Select>
        </FormControl>
      </Box>

      {/* Key Metrics */}
      <Grid container spacing={3} mb={4}>
        <Grid item xs={12} sm={6} md={4} lg={2.4}>
          <MetricCard
            title="Total Beds"
            value={metrics?.totalBeds || 0}
            icon={<Hotel />}
            color="primary"
            loading={metricsLoading}
            subtitle={`${metrics?.occupancyRate?.toFixed(0) || 0}% occupied`}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={4} lg={2.4}>
          <MetricCard
            title="Active Patients"
            value={realtimeStats?.activePatients || 0}
            icon={<People />}
            color="info"
            loading={metricsLoading}
            subtitle="Currently admitted"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={4} lg={2.4}>
          <MetricCard
            title="Critical Patients"
            value={realtimeStats?.criticalPatients || 0}
            icon={<Warning />}
            color="error"
            loading={metricsLoading}
            subtitle="Requires attention"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={4} lg={2.4}>
          <MetricCard
            title="Sepsis Alerts"
            value={realtimeStats?.activeSepsisAlerts || 0}
            icon={<Warning />}
            color="warning"
            loading={metricsLoading}
            subtitle="Active monitoring"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={4} lg={2.4}>
          <MetricCard
            title="Mortality Rate"
            value={`${metrics?.mortalityRate?.toFixed(1) || '0.0'}%`}
            icon={<TrendingUp />}
            color="success"
            loading={metricsLoading}
            subtitle="30-day average"
          />
        </Grid>
      </Grid>

      {/* Charts */}
      <Grid container spacing={3}>
        {/* Risk Distribution Pie Chart */}
        <Grid item xs={12} md={6} lg={4}>
          <Paper sx={{ p: 3, height: 400 }}>
            <Typography variant="h6" gutterBottom fontWeight={600}>
              Risk Distribution
            </Typography>
            <ResponsiveContainer width="100%" height="90%">
              <PieChart>
                <Pie
                  data={riskDistributionData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(0)}%`}
                  outerRadius={isMobile ? 60 : 80}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {riskDistributionData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>

        {/* Risk Trends Over Time */}
        <Grid item xs={12} md={12} lg={8}>
          <Paper sx={{ p: 3, height: 400 }}>
            <Typography variant="h6" gutterBottom fontWeight={600}>
              Risk Trends Over Time
            </Typography>
            {trendsLoading ? (
              <Box display="flex" justifyContent="center" alignItems="center" height="90%">
                <Typography color="text.secondary">Loading trends...</Typography>
              </Box>
            ) : (
              <ResponsiveContainer width="100%" height="90%">
                <LineChart data={trendChartData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="Total"
                    stroke={theme.palette.primary.main}
                    strokeWidth={2}
                    dot={{ r: 3 }}
                    name="Total Patients"
                  />
                  <Line
                    type="monotone"
                    dataKey="High"
                    stroke={RISK_COLORS.medium}
                    strokeWidth={2}
                    dot={{ r: 3 }}
                    name="High Risk"
                  />
                  <Line
                    type="monotone"
                    dataKey="Critical"
                    stroke={RISK_COLORS.high}
                    strokeWidth={2}
                    dot={{ r: 3 }}
                    name="Critical"
                  />
                </LineChart>
              </ResponsiveContainer>
            )}
          </Paper>
        </Grid>

        {/* Department Overview */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom fontWeight={600} mb={2}>
              Department Overview
            </Typography>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={departmentData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="patients" fill={theme.palette.primary.main} name="Total Patients" />
                <Bar dataKey="highRisk" fill={RISK_COLORS.high} name="High Risk" />
                <Bar dataKey="alerts" fill={theme.palette.warning.main} name="Active Alerts" />
              </BarChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>

        {/* Department Details Table */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom fontWeight={600} mb={2}>
              Department Statistics
            </Typography>
            <Box sx={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ borderBottom: `2px solid ${theme.palette.divider}` }}>
                    <th style={{ textAlign: 'left', padding: '12px', fontWeight: 600 }}>Department</th>
                    <th style={{ textAlign: 'center', padding: '12px', fontWeight: 600 }}>Patients</th>
                    <th style={{ textAlign: 'center', padding: '12px', fontWeight: 600 }}>High Risk</th>
                    <th style={{ textAlign: 'center', padding: '12px', fontWeight: 600 }}>Alerts</th>
                    <th style={{ textAlign: 'center', padding: '12px', fontWeight: 600 }}>Avg Risk Score</th>
                  </tr>
                </thead>
                <tbody>
                  {departmentData.map((dept, index) => (
                    <tr
                      key={dept.name}
                      style={{
                        borderBottom: `1px solid ${theme.palette.divider}`,
                        backgroundColor: index % 2 === 0 ? theme.palette.action.hover : 'transparent',
                      }}
                    >
                      <td style={{ padding: '12px' }}>
                        <Box display="flex" alignItems="center" gap={1}>
                          <LocalHospital fontSize="small" color="primary" />
                          <Typography variant="body2" fontWeight={500}>
                            {dept.name}
                          </Typography>
                        </Box>
                      </td>
                      <td style={{ textAlign: 'center', padding: '12px' }}>
                        <Typography variant="body2">{dept.patients}</Typography>
                      </td>
                      <td style={{ textAlign: 'center', padding: '12px' }}>
                        <Typography
                          variant="body2"
                          fontWeight={600}
                          color={dept.highRisk > 0 ? 'error.main' : 'text.secondary'}
                        >
                          {dept.highRisk}
                        </Typography>
                      </td>
                      <td style={{ textAlign: 'center', padding: '12px' }}>
                        <Typography
                          variant="body2"
                          fontWeight={600}
                          color={dept.alerts > 0 ? 'warning.main' : 'text.secondary'}
                        >
                          {dept.alerts}
                        </Typography>
                      </td>
                      <td style={{ textAlign: 'center', padding: '12px' }}>
                        <Typography variant="body2">{dept.avgRisk.toFixed(1)}</Typography>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default ExecutiveDashboard;
