import React, { useState, useEffect } from 'react';
import {
  Box,
  Container,
  AppBar,
  Toolbar,
  Typography,
  Tabs,
  Tab,
  IconButton,
  Badge,
  Avatar,
  Menu,
  MenuItem,
  Alert,
  Snackbar,
  useTheme,
  useMediaQuery,
  Drawer,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
} from '@mui/material';
import {
  Dashboard as DashboardIcon,
  LocalHospital,
  Person,
  Notifications,
  Menu as MenuIcon,
  Settings,
  Logout,
  Assessment,
} from '@mui/icons-material';
import { useQuery } from '@apollo/client';
import ExecutiveDashboard from './components/ExecutiveDashboard';
import ClinicalDashboard from './components/ClinicalDashboard';
import PatientDetailDashboard from './components/PatientDetailDashboard';
import QualityMetricsDashboard from './components/QualityMetricsDashboard';
import { GET_ACTIVE_ALERTS, Alert as AlertType } from './graphql/queries-fixed';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => (
  <div role="tabpanel" hidden={value !== index}>
    {value === index && <Box sx={{ py: 3 }}>{children}</Box>}
  </div>
);

const App: React.FC = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [activeTab, setActiveTab] = useState(0);
  const [selectedPatientId, setSelectedPatientId] = useState<string | null>(null);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [notificationAnchor, setNotificationAnchor] = useState<null | HTMLElement>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [snackbarMessage, setSnackbarMessage] = useState('');

  // Fetch active alerts (with automatic polling every 30 seconds)
  const { data: alertsData } = useQuery<{ sepsisAlerts: AlertType[] }>(GET_ACTIVE_ALERTS, {
    variables: { hospitalId: 'HOSPITAL-001', limit: 10 },
    pollInterval: 30000, // Poll every 30 seconds for new alerts
  });

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
    // Clear patient selection if navigating away from Patient Detail tab
    if (newValue !== 2) {
      setSelectedPatientId(null);
    }
  };

  const handlePatientSelect = (patientId: string) => {
    setSelectedPatientId(patientId);
    setActiveTab(2); // Switch to Patient Detail tab
  };

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleNotificationOpen = (event: React.MouseEvent<HTMLElement>) => {
    setNotificationAnchor(event.currentTarget);
  };

  const handleNotificationClose = () => {
    setNotificationAnchor(null);
  };

  const activeAlerts = alertsData?.sepsisAlerts || [];
  const unreadAlertCount = activeAlerts.length; // All alerts are considered unread

  const renderMobileMenu = () => (
    <Drawer
      anchor="left"
      open={mobileMenuOpen}
      onClose={() => setMobileMenuOpen(false)}
    >
      <List sx={{ width: 250 }}>
        <ListItem button onClick={() => { setActiveTab(0); setMobileMenuOpen(false); }}>
          <ListItemIcon><DashboardIcon /></ListItemIcon>
          <ListItemText primary="Executive" />
        </ListItem>
        <ListItem button onClick={() => { setActiveTab(1); setMobileMenuOpen(false); }}>
          <ListItemIcon><LocalHospital /></ListItemIcon>
          <ListItemText primary="Clinical" />
        </ListItem>
        {selectedPatientId && (
          <ListItem button onClick={() => { setActiveTab(2); setMobileMenuOpen(false); }}>
            <ListItemIcon><Person /></ListItemIcon>
            <ListItemText primary="Patient Detail" />
          </ListItem>
        )}
        <ListItem button onClick={() => { setActiveTab(3); setMobileMenuOpen(false); }}>
          <ListItemIcon><Assessment /></ListItemIcon>
          <ListItemText primary="Quality Metrics" />
        </ListItem>
      </List>
    </Drawer>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <AppBar position="sticky" elevation={2}>
        <Toolbar>
          {isMobile && (
            <IconButton
              edge="start"
              color="inherit"
              onClick={() => setMobileMenuOpen(true)}
              sx={{ mr: 2 }}
            >
              <MenuIcon />
            </IconButton>
          )}

          <LocalHospital sx={{ mr: 2 }} />
          <Typography variant="h6" component="div" sx={{ flexGrow: 1, fontWeight: 600 }}>
            {import.meta.env.VITE_HOSPITAL_NAME || 'CardioFit Clinical Dashboard'}
          </Typography>

          <IconButton
            color="inherit"
            onClick={handleNotificationOpen}
            sx={{ mr: 2 }}
          >
            <Badge badgeContent={unreadAlertCount} color="error">
              <Notifications />
            </Badge>
          </IconButton>

          <IconButton onClick={handleMenuOpen} sx={{ p: 0 }}>
            <Avatar sx={{ bgcolor: theme.palette.secondary.main }}>
              <Person />
            </Avatar>
          </IconButton>

          <Menu
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={handleMenuClose}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          >
            <MenuItem onClick={handleMenuClose}>
              <Settings sx={{ mr: 1 }} /> Settings
            </MenuItem>
            <MenuItem onClick={handleMenuClose}>
              <Logout sx={{ mr: 1 }} /> Logout
            </MenuItem>
          </Menu>

          <Menu
            anchorEl={notificationAnchor}
            open={Boolean(notificationAnchor)}
            onClose={handleNotificationClose}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            PaperProps={{ sx: { width: 350, maxHeight: 400 } }}
          >
            {activeAlerts.length > 0 ? (
              activeAlerts.map((alert: AlertType, index: number) => (
                <MenuItem key={`${alert.patientId}-${index}`} onClick={handleNotificationClose}>
                  <Box>
                    <Typography variant="subtitle2" color="text.primary">
                      Patient: {alert.patientId}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {alert.sepsisSeverity} Sepsis Alert - Stage: {alert.sepsisStage}
                    </Typography>
                    <Typography variant="body2" color="text.secondary" fontSize="0.85rem">
                      Risk: {alert.sepsisRisk.toFixed(1)}% | Department: {alert.departmentId}
                    </Typography>
                    <Typography variant="caption" color="text.disabled">
                      {new Date(alert.timestamp).toLocaleString()}
                    </Typography>
                  </Box>
                </MenuItem>
              ))
            ) : (
              <MenuItem disabled>
                <Typography variant="body2">No active alerts</Typography>
              </MenuItem>
            )}
          </Menu>
        </Toolbar>

        {!isMobile && (
          <Tabs
            value={activeTab}
            onChange={handleTabChange}
            textColor="inherit"
            indicatorColor="secondary"
            sx={{ borderTop: 1, borderColor: 'rgba(255,255,255,0.12)' }}
          >
            <Tab icon={<DashboardIcon />} label="Executive" />
            <Tab icon={<LocalHospital />} label="Clinical" />
            {selectedPatientId && <Tab icon={<Person />} label="Patient Detail" />}
            <Tab icon={<Assessment />} label="Quality Metrics" />
          </Tabs>
        )}
      </AppBar>

      {renderMobileMenu()}

      <Container maxWidth="xl" sx={{ flexGrow: 1 }}>
        <TabPanel value={activeTab} index={0}>
          <ExecutiveDashboard />
        </TabPanel>
        <TabPanel value={activeTab} index={1}>
          <ClinicalDashboard onPatientSelect={handlePatientSelect} />
        </TabPanel>
        {selectedPatientId && (
          <TabPanel value={activeTab} index={2}>
            <PatientDetailDashboard patientId={selectedPatientId} />
          </TabPanel>
        )}
        <TabPanel value={activeTab} index={3}>
          <QualityMetricsDashboard />
        </TabPanel>
      </Container>

      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={() => setSnackbarOpen(false)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert
          onClose={() => setSnackbarOpen(false)}
          severity="warning"
          variant="filled"
          sx={{ width: '100%' }}
        >
          {snackbarMessage}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default App;
