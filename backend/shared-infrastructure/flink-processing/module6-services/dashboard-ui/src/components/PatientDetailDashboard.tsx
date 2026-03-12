import React from 'react';
import {
  Grid,
  Paper,
  Typography,
  Box,
  Alert,
  Chip,
  Divider,
  Card,
  CardContent,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  useTheme,
  CircularProgress,
  Button,
} from '@mui/material';
import {
  Person,
  LocalHospital,
  Favorite,
  Thermostat,
  Speed,
  Warning,
  Medication,
  ArrowBack,
} from '@mui/icons-material';
import { useQuery } from '@apollo/client';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  AreaChart,
  Area,
} from 'recharts';
import {
  GET_PATIENT_DETAIL,
  PatientDetail,
  VitalSign,
} from '../graphql/queries-fixed';

interface PatientDetailDashboardProps {
  patientId: string;
}

const PatientDetailDashboard: React.FC<PatientDetailDashboardProps> = ({ patientId }) => {
  const theme = useTheme();

  // Fetch patient details with automatic polling every 30 seconds
  const { data, loading, error } = useQuery<{ patient: PatientDetail }>(
    GET_PATIENT_DETAIL,
    {
      variables: { patientId },
      pollInterval: 30000, // Poll every 30 seconds for updates
    }
  );

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={400}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mt: 2 }}>
        Failed to load patient details: {error.message}
      </Alert>
    );
  }

  if (!data?.patient) {
    return (
      <Alert severity="info" sx={{ mt: 2 }}>
        Patient not found
      </Alert>
    );
  }

  const patient = data.patient;

  const getRiskColor = (category: string) => {
    switch (category.toUpperCase()) {
      case 'HIGH':
        return theme.palette.error.main;
      case 'MEDIUM':
        return theme.palette.warning.main;
      case 'LOW':
        return theme.palette.success.main;
      default:
        return theme.palette.grey[500];
    }
  };

  const getVitalStatus = (status: string) => {
    switch (status.toLowerCase()) {
      case 'critical':
        return { color: 'error', icon: <Warning /> };
      case 'abnormal':
        return { color: 'warning', icon: <Warning /> };
      case 'normal':
        return { color: 'success', icon: null };
      default:
        return { color: 'default', icon: null };
    }
  };

  // Prepare risk history chart data
  const riskHistoryData = patient.riskHistory
    .slice(-20)
    .map((score) => ({
      time: new Date(score.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      score: score.riskScore,
    }));

  // Group vitals by type for chart
  const vitalsByType: Record<string, VitalSign[]> = {};
  patient.vitals.forEach((vital) => {
    if (!vitalsByType[vital.type]) {
      vitalsByType[vital.type] = [];
    }
    vitalsByType[vital.type].push(vital);
  });

  // Prepare vital signs chart data (last 24 hours)
  const vitalsChartData = patient.vitals.slice(-20).map((vital) => ({
    time: new Date(vital.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    [vital.type]: vital.value,
  }));

  // Consolidate vital signs for chart
  const consolidatedVitals = patient.vitals.reduce((acc, vital) => {
    const timeKey = new Date(vital.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    const existing = acc.find(v => v.time === timeKey);
    if (existing) {
      existing[vital.type] = vital.value;
    } else {
      acc.push({ time: timeKey, [vital.type]: vital.value });
    }
    return acc;
  }, [] as Array<Record<string, string | number>>);

  return (
    <Box>
      <Box display="flex" alignItems="center" gap={2} mb={3}>
        <Button
          startIcon={<ArrowBack />}
          onClick={() => window.history.back()}
          variant="outlined"
        >
          Back
        </Button>
        <Typography variant="h4" fontWeight={600}>
          Patient Details
        </Typography>
      </Box>

      {/* Patient Header */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Grid container spacing={3}>
          <Grid item xs={12} md={8}>
            <Box display="flex" alignItems="flex-start" gap={2}>
              <Box
                sx={{
                  width: 80,
                  height: 80,
                  borderRadius: 2,
                  backgroundColor: theme.palette.primary.light,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Person sx={{ fontSize: 48, color: theme.palette.primary.contrastText }} />
              </Box>
              <Box flexGrow={1}>
                <Typography variant="h5" fontWeight={600} gutterBottom>
                  {patient.name}
                </Typography>
                <Box display="flex" gap={2} flexWrap="wrap" mb={1}>
                  <Typography variant="body2" color="text.secondary">
                    ID: {patient.id}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Age: {patient.age}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Gender: {patient.gender}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Department: {patient.department}
                  </Typography>
                </Box>
                <Typography variant="body2" color="text.secondary">
                  Admitted: {new Date(patient.admissionDate).toLocaleDateString()}
                </Typography>
                <Typography variant="body2" color="text.secondary" mt={1}>
                  Diagnosis: {patient.diagnosis}
                </Typography>
              </Box>
            </Box>
          </Grid>
          <Grid item xs={12} md={4}>
            <Card
              sx={{
                backgroundColor: getRiskColor(patient.riskCategory) + '15',
                border: `2px solid ${getRiskColor(patient.riskCategory)}`,
              }}
            >
              <CardContent>
                <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                  Current Risk Assessment
                </Typography>
                <Typography variant="h3" fontWeight={700} color={getRiskColor(patient.riskCategory)}>
                  {patient.currentRiskScore.toFixed(1)}
                </Typography>
                <Chip
                  label={patient.riskCategory}
                  sx={{
                    mt: 1,
                    backgroundColor: getRiskColor(patient.riskCategory),
                    color: 'white',
                    fontWeight: 600,
                  }}
                />
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </Paper>

      {/* Risk Trend Chart */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Typography variant="h6" gutterBottom fontWeight={600}>
          Risk Score Trend
        </Typography>
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={riskHistoryData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" />
            <YAxis domain={[0, 100]} />
            <Tooltip />
            <Area
              type="monotone"
              dataKey="score"
              stroke={theme.palette.primary.main}
              fill={theme.palette.primary.light}
              strokeWidth={2}
            />
          </AreaChart>
        </ResponsiveContainer>
      </Paper>

      <Grid container spacing={3}>
        {/* Vital Signs */}
        <Grid item xs={12} lg={8}>
          <Paper sx={{ p: 3, mb: 3 }}>
            <Typography variant="h6" gutterBottom fontWeight={600} mb={2}>
              Vital Signs
            </Typography>
            <Grid container spacing={2} mb={3}>
              {patient.vitals.slice(-4).map((vital, index) => {
                const status = getVitalStatus(vital.status);
                return (
                  <Grid item xs={12} sm={6} key={index}>
                    <Card variant="outlined">
                      <CardContent>
                        <Box display="flex" justifyContent="space-between" alignItems="center">
                          <Box>
                            <Typography variant="subtitle2" color="text.secondary">
                              {vital.type}
                            </Typography>
                            <Typography variant="h5" fontWeight={600}>
                              {vital.value} <Typography component="span" variant="body2">{vital.unit}</Typography>
                            </Typography>
                            <Chip
                              label={vital.status}
                              size="small"
                              color={status.color as any}
                              icon={status.icon}
                              sx={{ mt: 1 }}
                            />
                          </Box>
                          {vital.type === 'Heart Rate' && <Favorite fontSize="large" color="error" />}
                          {vital.type === 'Temperature' && <Thermostat fontSize="large" color="warning" />}
                          {vital.type === 'Blood Pressure' && <Speed fontSize="large" color="primary" />}
                        </Box>
                        <Typography variant="caption" color="text.disabled" display="block" mt={1}>
                          {new Date(vital.timestamp).toLocaleString()}
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                );
              })}
            </Grid>

            {consolidatedVitals.length > 0 && (
              <>
                <Divider sx={{ my: 2 }} />
                <Typography variant="subtitle2" gutterBottom fontWeight={600}>
                  Vital Signs Trend (Last 24 Hours)
                </Typography>
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={consolidatedVitals.slice(-20)}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="time" />
                    <YAxis />
                    <Tooltip />
                    <Legend />
                    {Object.keys(vitalsByType).map((type, index) => {
                      const colors = [
                        theme.palette.error.main,
                        theme.palette.primary.main,
                        theme.palette.success.main,
                        theme.palette.warning.main,
                      ];
                      return (
                        <Line
                          key={type}
                          type="monotone"
                          dataKey={type}
                          stroke={colors[index % colors.length]}
                          strokeWidth={2}
                          dot={{ r: 3 }}
                        />
                      );
                    })}
                  </LineChart>
                </ResponsiveContainer>
              </>
            )}
          </Paper>

          {/* Active Alerts */}
          {patient.alerts.length > 0 && (
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom fontWeight={600} mb={2}>
                Active Alerts ({patient.alerts.length})
              </Typography>
              <List>
                {patient.alerts.map((alert, index) => (
                  <React.Fragment key={alert.id}>
                    <ListItem
                      sx={{
                        backgroundColor:
                          alert.severity === 'CRITICAL'
                            ? theme.palette.error.light + '15'
                            : theme.palette.warning.light + '15',
                        borderRadius: 1,
                        mb: 1,
                      }}
                    >
                      <ListItemIcon>
                        <Warning color={alert.severity === 'CRITICAL' ? 'error' : 'warning'} />
                      </ListItemIcon>
                      <ListItemText
                        primary={
                          <Box display="flex" alignItems="center" gap={1}>
                            <Typography variant="subtitle2" fontWeight={600}>
                              {alert.message}
                            </Typography>
                            <Chip
                              label={alert.severity}
                              size="small"
                              color={alert.severity === 'CRITICAL' ? 'error' : 'warning'}
                            />
                          </Box>
                        }
                        secondary={new Date(alert.timestamp).toLocaleString()}
                      />
                    </ListItem>
                    {index < patient.alerts.length - 1 && <Divider />}
                  </React.Fragment>
                ))}
              </List>
            </Paper>
          )}
        </Grid>

        {/* Medications and Risk Factors */}
        <Grid item xs={12} lg={4}>
          {/* Current Medications */}
          <Paper sx={{ p: 3, mb: 3 }}>
            <Typography variant="h6" gutterBottom fontWeight={600} mb={2}>
              <Medication sx={{ mr: 1, verticalAlign: 'middle' }} />
              Current Medications
            </Typography>
            <List dense>
              {patient.medications.map((med, index) => (
                <React.Fragment key={index}>
                  <ListItem>
                    <ListItemText
                      primary={
                        <Typography variant="body2" fontWeight={600}>
                          {med.name}
                        </Typography>
                      }
                      secondary={
                        <>
                          <Typography variant="caption" display="block">
                            Dosage: {med.dosage}
                          </Typography>
                          <Typography variant="caption" display="block">
                            Frequency: {med.frequency}
                          </Typography>
                          <Typography variant="caption" display="block" color="text.disabled">
                            Started: {new Date(med.startDate).toLocaleDateString()}
                          </Typography>
                        </>
                      }
                    />
                  </ListItem>
                  {index < patient.medications.length - 1 && <Divider />}
                </React.Fragment>
              ))}
            </List>
          </Paper>

          {/* Risk Factors */}
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom fontWeight={600} mb={2}>
              <Warning sx={{ mr: 1, verticalAlign: 'middle' }} />
              Risk Factors
            </Typography>
            <Box display="flex" flexWrap="wrap" gap={1}>
              {patient.riskHistory[0]?.factors.map((factor, index) => (
                <Chip
                  key={index}
                  label={factor}
                  color="error"
                  variant="outlined"
                  size="small"
                />
              ))}
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default PatientDetailDashboard;
