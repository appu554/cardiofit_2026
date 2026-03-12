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
} from '@mui/material';
import {
  CheckCircle,
  LocalHospital,
  AccessTime,
} from '@mui/icons-material';
import { useQuery } from '@apollo/client';
import MetricCard from './MetricCard';
import {
  GET_QUALITY_METRICS,
  QualityMetric,
} from '../graphql/queries-fixed';

const QualityMetricsDashboard: React.FC = () => {
  const theme = useTheme();
  const [period, setPeriod] = useState('30');
  const [selectedDepartment, setSelectedDepartment] = useState<string | null>(null);

  const { data, loading, error } = useQuery<{
    qualityMetrics: QualityMetric[];
  }>(GET_QUALITY_METRICS, {
    variables: {
      hospitalId: 'HOSPITAL-001',
      departmentId: selectedDepartment,
      period: period,
    },
    pollInterval: 30000,
  });

  const handlePeriodChange = (event: SelectChangeEvent) => {
    setPeriod(event.target.value);
  };

  const handleDepartmentChange = (event: SelectChangeEvent) => {
    setSelectedDepartment(event.target.value || null);
  };

  if (error) {
    return (
      <Alert severity="error" sx={{ mt: 2 }}>
        Failed to load quality metrics: {error.message}
      </Alert>
    );
  }

  const metrics = data?.qualityMetrics?.[0]; // Get first metric

  // Calculate compliance rate percentage
  const complianceRate = metrics?.sepsisComplianceRate || 0;
  const complianceColor = complianceRate >= 90 ? 'success' : complianceRate >= 80 ? 'warning' : 'error';

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3} flexWrap="wrap" gap={2}>
        <Typography variant="h4" fontWeight={600}>
          Quality Metrics Dashboard
        </Typography>
        <Box display="flex" gap={2}>
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <Select value={selectedDepartment || ''} onChange={handleDepartmentChange} displayEmpty>
              <MenuItem value="">All Departments</MenuItem>
              <MenuItem value="ICU">ICU</MenuItem>
              <MenuItem value="Emergency">Emergency</MenuItem>
              <MenuItem value="Cardiology">Cardiology</MenuItem>
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <Select value={period} onChange={handlePeriodChange}>
              <MenuItem value="7">Last 7 Days</MenuItem>
              <MenuItem value="30">Last 30 Days</MenuItem>
              <MenuItem value="90">Last 90 Days</MenuItem>
            </Select>
          </FormControl>
        </Box>
      </Box>

      {/* Key Performance Indicators */}
      <Grid container spacing={3} mb={4}>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Sepsis Bundle Compliance"
            value={`${metrics?.sepsisBundleCompliance?.toFixed(1) || '0.0'}%`}
            icon={<CheckCircle />}
            color={complianceColor}
            loading={loading}
            subtitle="Overall compliance"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Compliance Rate"
            value={`${complianceRate.toFixed(1)}%`}
            icon={<CheckCircle />}
            color={complianceColor}
            loading={loading}
            subtitle={`${metrics?.sepsisCompliantEncounters || 0} / ${metrics?.sepsisEncounters || 0} cases`}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Avg Time to Antibiotic"
            value={`${metrics?.avgTimeToAntibiotic?.toFixed(0) || '0'} min`}
            icon={<AccessTime />}
            color="info"
            loading={loading}
            subtitle="From admission"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Total Encounters"
            value={metrics?.sepsisEncounters || 0}
            icon={<LocalHospital />}
            color="primary"
            loading={loading}
            subtitle="Sepsis patients"
          />
        </Grid>
      </Grid>

      {/* Compliance Details */}
      <Grid container spacing={3}>
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom fontWeight={600} mb={3}>
              Sepsis Bundle Compliance Details
            </Typography>
            {loading ? (
              <Box display="flex" justifyContent="center" alignItems="center" height={200}>
                <Typography color="text.secondary">Loading quality metrics...</Typography>
              </Box>
            ) : metrics ? (
              <Box>
                <Grid container spacing={3}>
                  <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 2, backgroundColor: theme.palette.background.default }}>
                      <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                        Compliance Summary
                      </Typography>
                      <Box display="flex" justifyContent="space-between" alignItems="center" mt={2}>
                        <Typography variant="body2">Total Sepsis Encounters:</Typography>
                        <Typography variant="h6" fontWeight={600}>
                          {metrics.sepsisEncounters}
                        </Typography>
                      </Box>
                      <Box display="flex" justifyContent="space-between" alignItems="center" mt={1}>
                        <Typography variant="body2">Compliant Cases:</Typography>
                        <Typography variant="h6" fontWeight={600} color="success.main">
                          {metrics.sepsisCompliantEncounters}
                        </Typography>
                      </Box>
                      <Box display="flex" justifyContent="space-between" alignItems="center" mt={1}>
                        <Typography variant="body2">Non-Compliant Cases:</Typography>
                        <Typography variant="h6" fontWeight={600} color="error.main">
                          {metrics.sepsisEncounters - metrics.sepsisCompliantEncounters}
                        </Typography>
                      </Box>
                    </Paper>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 2, backgroundColor: theme.palette.background.default }}>
                      <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                        Performance Metrics
                      </Typography>
                      <Box display="flex" justifyContent="space-between" alignItems="center" mt={2}>
                        <Typography variant="body2">Bundle Compliance:</Typography>
                        <Typography variant="h6" fontWeight={600}>
                          {metrics.sepsisBundleCompliance.toFixed(1)}%
                        </Typography>
                      </Box>
                      <Box display="flex" justifyContent="space-between" alignItems="center" mt={1}>
                        <Typography variant="body2">Compliance Rate:</Typography>
                        <Typography variant="h6" fontWeight={600}>
                          {metrics.sepsisComplianceRate.toFixed(1)}%
                        </Typography>
                      </Box>
                      <Box display="flex" justifyContent="space-between" alignItems="center" mt={1}>
                        <Typography variant="body2">Avg Time to Antibiotic:</Typography>
                        <Typography variant="h6" fontWeight={600}>
                          {metrics.avgTimeToAntibiotic.toFixed(0)} minutes
                        </Typography>
                      </Box>
                    </Paper>
                  </Grid>
                </Grid>

                <Box mt={3}>
                  <Typography variant="body2" color="text.secondary">
                    Last Updated: {new Date(metrics.timestamp).toLocaleString()}
                  </Typography>
                </Box>
              </Box>
            ) : (
              <Box display="flex" justifyContent="center" alignItems="center" height={200}>
                <Typography color="text.secondary">No quality metrics available</Typography>
              </Box>
            )}
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default QualityMetricsDashboard;
