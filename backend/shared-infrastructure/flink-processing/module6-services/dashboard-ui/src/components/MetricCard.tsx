import React from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  CircularProgress,
  Chip,
  useTheme,
} from '@mui/material';
import {
  TrendingUp,
  TrendingDown,
  TrendingFlat,
} from '@mui/icons-material';

interface MetricCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: React.ReactNode;
  trend?: 'up' | 'down' | 'flat';
  trendValue?: string;
  color?: 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info';
  loading?: boolean;
  onClick?: () => void;
}

const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  subtitle,
  icon,
  trend,
  trendValue,
  color = 'primary',
  loading = false,
  onClick,
}) => {
  const theme = useTheme();

  const getTrendIcon = () => {
    switch (trend) {
      case 'up':
        return <TrendingUp fontSize="small" />;
      case 'down':
        return <TrendingDown fontSize="small" />;
      case 'flat':
        return <TrendingFlat fontSize="small" />;
      default:
        return null;
    }
  };

  const getTrendColor = () => {
    switch (trend) {
      case 'up':
        return theme.palette.success.main;
      case 'down':
        return theme.palette.error.main;
      case 'flat':
        return theme.palette.grey[500];
      default:
        return theme.palette.text.secondary;
    }
  };

  return (
    <Card
      sx={{
        height: '100%',
        cursor: onClick ? 'pointer' : 'default',
        transition: 'transform 0.2s, box-shadow 0.2s',
        '&:hover': onClick
          ? {
              transform: 'translateY(-4px)',
              boxShadow: theme.shadows[8],
            }
          : {},
      }}
      onClick={onClick}
    >
      <CardContent>
        <Box display="flex" alignItems="flex-start" justifyContent="space-between" mb={2}>
          <Typography
            variant="subtitle2"
            color="text.secondary"
            sx={{ fontWeight: 500, textTransform: 'uppercase', letterSpacing: 0.5 }}
          >
            {title}
          </Typography>
          {icon && (
            <Box
              sx={{
                color: theme.palette[color].main,
                backgroundColor: theme.palette[color].light + '20',
                borderRadius: 1,
                p: 0.5,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              {icon}
            </Box>
          )}
        </Box>

        {loading ? (
          <Box display="flex" justifyContent="center" py={2}>
            <CircularProgress size={40} />
          </Box>
        ) : (
          <>
            <Typography
              variant="h4"
              component="div"
              sx={{
                fontWeight: 700,
                color: theme.palette[color].main,
                mb: 1,
              }}
            >
              {value}
            </Typography>

            {subtitle && (
              <Typography variant="body2" color="text.secondary" mb={1}>
                {subtitle}
              </Typography>
            )}

            {trend && trendValue && (
              <Box display="flex" alignItems="center" gap={1} mt={1}>
                <Chip
                  icon={getTrendIcon()}
                  label={trendValue}
                  size="small"
                  sx={{
                    backgroundColor: getTrendColor() + '20',
                    color: getTrendColor(),
                    fontWeight: 600,
                    '& .MuiChip-icon': {
                      color: getTrendColor(),
                    },
                  }}
                />
              </Box>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
};

export default MetricCard;
