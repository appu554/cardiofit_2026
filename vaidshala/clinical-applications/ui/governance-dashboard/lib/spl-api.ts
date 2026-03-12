// ============================================================================
// SPL Fact Review API Client
// Queries derived_facts + completeness_reports via KB-0 governance endpoints
// ============================================================================

import axios, { AxiosInstance } from 'axios';
import type {
  CompletenessReport,
  SPLDerivedFact,
  DrugTriageState,
  SPLFactType,
  GovernanceStatus,
  ExtractionMethod,
  FactReviewDecision,
  DrugSignOff,
} from '@/types/spl-review';
import type { PaginatedResponse } from '@/types/governance';

// ============================================================================
// API Client Configuration
// ============================================================================

// SPL API routes live under /api/v2/spl/ — separate from the governance queue at /api/v2/governance/
const SPL_API_BASE = process.env.NEXT_PUBLIC_KB0_API_URL
  ? process.env.NEXT_PUBLIC_KB0_API_URL.replace(/\/governance\/?$/, '')
  : '/api/v2';

const apiClient: AxiosInstance = axios.create({
  baseURL: SPL_API_BASE,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
});

// Auth interceptor — skip in dev mode to avoid cookie explosion
apiClient.interceptors.request.use(async (config) => {
  if (typeof window !== 'undefined' && process.env.NODE_ENV !== 'development') {
    try {
      const { getAccessToken } = await import('@auth0/nextjs-auth0/client');
      const token = await getAccessToken();
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    } catch {
      // Auth0 session not available
    }
  }
  return config;
});

// ============================================================================
// Completeness Reports API
// ============================================================================

export const completenessApi = {
  /**
   * Get all completeness reports (latest per drug)
   */
  async getAll(): Promise<CompletenessReport[]> {
    const { data } = await apiClient.get<{ items: CompletenessReport[] }>(
      '/spl/completeness'
    );
    return data.items || [];
  },

  /**
   * Get completeness report for a specific drug
   */
  async getByDrug(drugName: string): Promise<CompletenessReport | null> {
    const { data } = await apiClient.get<CompletenessReport>(
      `/spl/completeness/${encodeURIComponent(drugName)}`
    );
    return data || null;
  },
};

// ============================================================================
// SPL Derived Facts API
// ============================================================================

export const splFactsApi = {
  /**
   * Get derived facts for a drug, optionally filtered by type/status
   */
  async getByDrug(
    drugName: string,
    filters?: {
      factType?: SPLFactType;
      status?: GovernanceStatus;
      method?: ExtractionMethod;
    },
    page = 1,
    pageSize = 50
  ): Promise<PaginatedResponse<SPLDerivedFact>> {
    const params = new URLSearchParams();
    params.set('drug', drugName);
    if (filters?.factType) params.set('factType', filters.factType);
    if (filters?.status) params.set('status', filters.status);
    if (filters?.method) params.set('method', filters.method);
    params.set('page', String(page));
    params.set('pageSize', String(pageSize));

    const { data } = await apiClient.get<PaginatedResponse<SPLDerivedFact>>(
      `/spl/facts?${params}`
    );
    return data;
  },

  /**
   * Get a single fact by ID
   */
  async getById(factId: string): Promise<SPLDerivedFact> {
    const { data } = await apiClient.get<SPLDerivedFact>(`/spl/facts/${factId}`);
    return data;
  },

  /**
   * Get facts pending review for a drug (the review queue)
   */
  async getPendingReview(drugName: string): Promise<SPLDerivedFact[]> {
    const { data } = await apiClient.get<{ items: SPLDerivedFact[] }>(
      `/spl/facts/pending/${encodeURIComponent(drugName)}`
    );
    return data.items || [];
  },

  /**
   * Get random sample of auto-approved facts for spot-check
   */
  async getAutoApprovedSample(
    drugName: string,
    sampleSize = 10
  ): Promise<SPLDerivedFact[]> {
    const { data } = await apiClient.get<{ items: SPLDerivedFact[] }>(
      `/spl/facts/sample/${encodeURIComponent(drugName)}?size=${sampleSize}`
    );
    return data.items || [];
  },

  /**
   * Submit review decision for a fact
   */
  async submitDecision(decision: FactReviewDecision): Promise<void> {
    await apiClient.post(`/spl/facts/${decision.factId}/review`, decision);
  },

  /**
   * Get SPL section HTML for source panel
   */
  async getSectionHtml(
    sourceDocumentId: string,
    sectionCode: string
  ): Promise<string> {
    const { data } = await apiClient.get<{ html: string }>(
      `/spl/source/${sourceDocumentId}/section/${sectionCode}`
    );
    return data.html || '';
  },
};

// ============================================================================
// Triage API (aggregated drug-level view)
// ============================================================================

export const triageApi = {
  /**
   * Get triage dashboard data — all drugs with their completeness + fact summaries.
   * This endpoint aggregates completeness_reports + derived_facts counts.
   */
  async getDashboard(): Promise<DrugTriageState[]> {
    const { data } = await apiClient.get<{ items: DrugTriageState[] }>(
      '/spl/triage'
    );
    return data.items || [];
  },

  /**
   * Set drug disposition (REVIEW, INVESTIGATE, OUT_OF_SCOPE)
   */
  async setDisposition(
    drugName: string,
    disposition: DrugTriageState['disposition'],
    note?: string
  ): Promise<void> {
    await apiClient.post(`/spl/triage/${encodeURIComponent(drugName)}/disposition`, {
      disposition,
      note,
    });
  },
};

// ============================================================================
// Drug Sign-Off API
// ============================================================================

export const signOffApi = {
  /**
   * Submit drug sign-off attestation
   */
  async submit(signOff: DrugSignOff): Promise<void> {
    await apiClient.post(
      `/spl/signoff/${encodeURIComponent(signOff.drugName)}`,
      signOff
    );
  },

  /**
   * Get existing sign-off for a drug (if any)
   */
  async get(drugName: string): Promise<DrugSignOff | null> {
    try {
      const { data } = await apiClient.get<DrugSignOff>(
        `/spl/signoff/${encodeURIComponent(drugName)}`
      );
      return data;
    } catch {
      return null;
    }
  },
};

// ============================================================================
// Export consolidated API
// ============================================================================

export const splReviewApi = {
  completeness: completenessApi,
  facts: splFactsApi,
  triage: triageApi,
  signOff: signOffApi,
};

export default splReviewApi;
