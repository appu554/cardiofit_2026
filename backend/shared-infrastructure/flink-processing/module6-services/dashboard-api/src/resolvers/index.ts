import { hospitalKpisResolvers } from './hospital-kpis.resolver';
import { departmentMetricsResolvers } from './department-metrics.resolver';
import { patientRiskResolvers } from './patient-risk.resolver';
import { sepsisSurveillanceResolvers } from './sepsis-surveillance.resolver';
import { qualityMetricsResolvers } from './quality-metrics.resolver';
import { dashboardResolvers } from './dashboard.resolver';
import { GraphQLScalarType, Kind } from 'graphql';

// Custom scalar for DateTime
const dateTimeScalar = new GraphQLScalarType({
  name: 'DateTime',
  description: 'DateTime custom scalar type',
  serialize(value: any) {
    if (value instanceof Date) {
      return value.toISOString();
    }
    return value;
  },
  parseValue(value: any) {
    return new Date(value);
  },
  parseLiteral(ast) {
    if (ast.kind === Kind.STRING) {
      return new Date(ast.value);
    }
    return null;
  },
});

// Custom scalar for JSON
const jsonScalar = new GraphQLScalarType({
  name: 'JSON',
  description: 'JSON custom scalar type',
  serialize(value: any) {
    return value;
  },
  parseValue(value: any) {
    return value;
  },
  parseLiteral(ast) {
    if (ast.kind === Kind.OBJECT) {
      return ast;
    }
    return null;
  },
});

// Merge all resolvers
export const resolvers = {
  DateTime: dateTimeScalar,
  JSON: jsonScalar,

  Query: {
    ...hospitalKpisResolvers.Query,
    ...departmentMetricsResolvers.Query,
    ...patientRiskResolvers.Query,
    ...sepsisSurveillanceResolvers.Query,
    ...qualityMetricsResolvers.Query,
    ...dashboardResolvers.Query,
  },

  HospitalKPIs: hospitalKpisResolvers.HospitalKPIs,
  DepartmentMetrics: departmentMetricsResolvers.DepartmentMetrics,
  PatientRiskProfile: patientRiskResolvers.PatientRiskProfile,
  SepsisSurveillance: sepsisSurveillanceResolvers.SepsisSurveillance,
  QualityMetrics: qualityMetricsResolvers.QualityMetrics,
};
