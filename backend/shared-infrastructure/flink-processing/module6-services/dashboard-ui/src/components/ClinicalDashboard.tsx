import React, { useState, useMemo } from 'react';
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
  Chip,
  IconButton,
  TextField,
  InputAdornment,
  Button,
  useTheme,
} from '@mui/material';
import {
  Search,
  FilterList,
  Refresh,
  Person,
  Warning,
  CheckCircle,
} from '@mui/icons-material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { useQuery } from '@apollo/client';
import MetricCard from './MetricCard';
import {
  GET_HIGH_RISK_PATIENTS,
  GET_ACTIVE_ALERTS,
  Patient,
  Alert as AlertType,
} from '../graphql/queries-fixed';

interface ClinicalDashboardProps {
  onPatientSelect: (patientId: string) => void;
}

const ClinicalDashboard: React.FC<ClinicalDashboardProps> = ({ onPatientSelect }) => {
  const theme = useTheme();
  const [selectedDepartment, setSelectedDepartment] = useState('ICU');
  const [searchQuery, setSearchQuery] = useState('');
  const [riskFilter, setRiskFilter] = useState<string>('all');

  const { data: patientsData, loading: patientsLoading, error: patientsError, refetch } = useQuery<{
    highRiskPatients: Patient[];
  }>(GET_HIGH_RISK_PATIENTS, {
    variables: { hospitalId: 'HOSPITAL-001', limit: 100 },
    pollInterval: 30000,
  });

  const { data: alertsData } = useQuery<{ sepsisAlerts: AlertType[] }>(GET_ACTIVE_ALERTS, {
    variables: { hospitalId: 'HOSPITAL-001', limit: 5 },
    pollInterval: 30000,
  });

  const handleDepartmentChange = (event: SelectChangeEvent) => {
    setSelectedDepartment(event.target.value);
  };

  const handleRiskFilterChange = (event: SelectChangeEvent) => {
    setRiskFilter(event.target.value);
  };

  const allPatients = patientsData?.highRiskPatients || [];
  const alerts = alertsData?.sepsisAlerts || [];

  // Filter patients by selected department first
  const departmentPatients = useMemo(() => {
    return allPatients.filter((patient) =>
      patient.departmentId === selectedDepartment
    );
  }, [allPatients, selectedDepartment]);

  // Filter and search patients
  const filteredPatients = useMemo(() => {
    return departmentPatients.filter((patient) => {
      const matchesSearch =
        patient.patientId?.toLowerCase().includes(searchQuery.toLowerCase());

      const matchesRisk =
        riskFilter === 'all' ||
        patient.riskCategory?.toLowerCase() === riskFilter.toLowerCase() ||
        patient.riskLevel?.toLowerCase() === riskFilter.toLowerCase();

      return matchesSearch && matchesRisk;
    });
  }, [departmentPatients, searchQuery, riskFilter]);

  // Calculate department metrics
  const departmentMetrics = useMemo(() => {
    const highRisk = departmentPatients.filter((p) =>
      p.riskLevel === 'HIGH' || p.riskCategory === 'HIGH'
    ).length;
    const mediumRisk = departmentPatients.filter((p) =>
      p.riskLevel === 'MODERATE' || p.riskCategory === 'MODERATE' || p.riskCategory === 'MEDIUM'
    ).length;
    const lowRisk = departmentPatients.filter((p) =>
      p.riskLevel === 'LOW' || p.riskCategory === 'LOW'
    ).length;
    const criticalCount = departmentPatients.filter((p) => p.riskLevel === 'CRITICAL' || p.isCritical).length;
    const totalAlerts = highRisk + criticalCount;

    return { highRisk, mediumRisk, lowRisk, totalAlerts };
  }, [departmentPatients]);

  const getRiskColor = (category: string | undefined) => {
    switch (category?.toUpperCase()) {
      case 'HIGH':
        return 'error';
      case 'MEDIUM':
        return 'warning';
      case 'LOW':
        return 'success';
      default:
        return 'default';
    }
  };

  const columns: GridColDef[] = [
    {
      field: 'patientId',
      headerName: 'Patient ID',
      width: 150,
      renderCell: (params: GridRenderCellParams<Patient>) => (
        <Box display="flex" alignItems="center" gap={1}>
          <Person fontSize="small" color="action" />
          <Typography variant="body2">{params.value}</Typography>
        </Box>
      ),
    },
    {
      field: 'departmentId',
      headerName: 'Department',
      width: 120,
    },
    {
      field: 'age',
      headerName: 'Age',
      width: 80,
      align: 'center',
      headerAlign: 'center',
    },
    {
      field: 'gender',
      headerName: 'Gender',
      width: 100,
    },
    {
      field: 'overallRiskScore',
      headerName: 'Risk Score',
      width: 120,
      align: 'center',
      headerAlign: 'center',
      renderCell: (params: GridRenderCellParams<Patient>) => (
        <Typography variant="body2" fontWeight={600}>
          {params.value?.toFixed(1) || 'N/A'}
        </Typography>
      ),
    },
    {
      field: 'riskCategory',
      headerName: 'Risk Level',
      width: 130,
      renderCell: (params: GridRenderCellParams<Patient>) => (
        <Chip
          label={params.value || 'Unknown'}
          color={getRiskColor(params.value as string)}
          size="small"
          sx={{ fontWeight: 600 }}
        />
      ),
    },
    {
      field: 'alertCount',
      headerName: 'Alerts',
      width: 100,
      align: 'center',
      headerAlign: 'center',
      renderCell: (params: GridRenderCellParams<Patient>) => {
        const alertCount = params.value as number;
        return alertCount > 0 ? (
          <Chip
            icon={<Warning />}
            label={alertCount}
            color="warning"
            size="small"
            sx={{ fontWeight: 600 }}
          />
        ) : (
          <CheckCircle color="success" fontSize="small" />
        );
      },
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 150,
      sortable: false,
      renderCell: (params: GridRenderCellParams<Patient>) => (
        <Button
          variant="outlined"
          size="small"
          onClick={() => onPatientSelect(params.row.patientId)}
        >
          View Details
        </Button>
      ),
    },
  ];

  if (patientsError) {
    return (
      <Alert severity="error" sx={{ mt: 2 }}>
        Failed to load patients: {patientsError.message}
      </Alert>
    );
  }

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3} flexWrap="wrap" gap={2}>
        <Typography variant="h4" fontWeight={600}>
          Clinical Dashboard - {selectedDepartment}
        </Typography>
        <Box display="flex" gap={2} alignItems="center">
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <Select value={selectedDepartment} onChange={handleDepartmentChange}>
              <MenuItem value="ICU">ICU</MenuItem>
              <MenuItem value="Emergency">Emergency</MenuItem>
              <MenuItem value="Cardiology">Cardiology</MenuItem>
              <MenuItem value="Surgery">Surgery</MenuItem>
              <MenuItem value="Pediatrics">Pediatrics</MenuItem>
            </Select>
          </FormControl>
          <IconButton color="primary" onClick={() => refetch()}>
            <Refresh />
          </IconButton>
        </Box>
      </Box>

      {/* Department Metrics */}
      <Grid container spacing={3} mb={4}>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Total Patients"
            value={departmentPatients.length}
            icon={<Person />}
            color="primary"
            loading={patientsLoading}
            subtitle={`In ${selectedDepartment}`}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="High Risk"
            value={departmentMetrics.highRisk}
            icon={<Warning />}
            color="error"
            loading={patientsLoading}
            subtitle="Critical attention"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Medium Risk"
            value={departmentMetrics.mediumRisk}
            icon={<Warning />}
            color="warning"
            loading={patientsLoading}
            subtitle="Monitor closely"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <MetricCard
            title="Active Alerts"
            value={departmentMetrics.totalAlerts}
            icon={<Warning />}
            color="info"
            loading={patientsLoading}
            subtitle="Unacknowledged"
          />
        </Grid>
      </Grid>

      {/* Recent Alerts */}
      {alerts.length > 0 && (
        <Paper sx={{ p: 3, mb: 3, backgroundColor: theme.palette.warning.light + '15' }}>
          <Typography variant="h6" gutterBottom fontWeight={600} mb={2}>
            Recent Alerts
          </Typography>
          <Grid container spacing={2}>
            {alerts.slice(0, 3).map((alert, index) => (
              <Grid item xs={12} key={`${alert.patientId}-${index}`}>
                <Box
                  sx={{
                    p: 2,
                    borderRadius: 1,
                    backgroundColor: theme.palette.background.paper,
                    border: `1px solid ${theme.palette.divider}`,
                  }}
                >
                  <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                    <Box>
                      <Typography variant="subtitle2" fontWeight={600} color="text.primary">
                        Patient: {alert.patientId} - {alert.sepsisSeverity}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        {alert.sepsisStage} Stage - Risk: {alert.sepsisRisk.toFixed(1)}%
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Bundle Status: {alert.onBundle ? 'On Bundle' : 'Not on Bundle'}
                      </Typography>
                    </Box>
                    <Typography variant="caption" color="text.disabled">
                      {new Date(alert.timestamp).toLocaleTimeString()}
                    </Typography>
                  </Box>
                </Box>
              </Grid>
            ))}
          </Grid>
        </Paper>
      )}

      {/* Patient List */}
      <Paper sx={{ p: 3 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3} flexWrap="wrap" gap={2}>
          <Typography variant="h6" fontWeight={600}>
            Patient List ({filteredPatients.length})
          </Typography>
          <Box display="flex" gap={2} flexWrap="wrap">
            <TextField
              size="small"
              placeholder="Search patients..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <Search />
                  </InputAdornment>
                ),
              }}
              sx={{ minWidth: 250 }}
            />
            <FormControl size="small" sx={{ minWidth: 150 }}>
              <Select
                value={riskFilter}
                onChange={handleRiskFilterChange}
                startAdornment={
                  <InputAdornment position="start">
                    <FilterList />
                  </InputAdornment>
                }
              >
                <MenuItem value="all">All Risk Levels</MenuItem>
                <MenuItem value="high">High Risk</MenuItem>
                <MenuItem value="medium">Medium Risk</MenuItem>
                <MenuItem value="low">Low Risk</MenuItem>
              </Select>
            </FormControl>
          </Box>
        </Box>

        <Box sx={{ height: 600, width: '100%' }}>
          <DataGrid
            rows={filteredPatients}
            columns={columns}
            loading={patientsLoading}
            pageSizeOptions={[10, 25, 50]}
            initialState={{
              pagination: { paginationModel: { pageSize: 10 } },
              sorting: { sortModel: [{ field: 'overallRiskScore', sort: 'desc' }] },
            }}
            disableRowSelectionOnClick
            sx={{
              '& .MuiDataGrid-row:hover': {
                cursor: 'pointer',
                backgroundColor: theme.palette.action.hover,
              },
            }}
          />
        </Box>
      </Paper>
    </Box>
  );
};

export default ClinicalDashboard;
