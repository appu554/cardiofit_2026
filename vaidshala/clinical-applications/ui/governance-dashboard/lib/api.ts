// ============================================================================
// KB-0 Governance API Service
// Communicates with KB-0 REST API v2 endpoints
// ============================================================================

import axios, { AxiosInstance, AxiosError } from 'axios';
import type {
  ClinicalFact,
  QueueItem,
  ReviewRequest,
  ReviewDecision,
  ConflictGroup,
  GovernanceMetrics,
  DashboardData,
  AuditEvent,
  ExecutorStatus,
  ApiResponse,
  PaginatedResponse,
  QueueFilters,
  SortOptions,
} from '@/types/governance';

// ============================================================================
// API Client Configuration
// ============================================================================

const API_BASE_URL = process.env.NEXT_PUBLIC_KB0_API_URL || '/api/v2/governance';

const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for Auth0 token
// In dev mode, skip token fetch to avoid __txn_ cookie explosion → HTTP 431
apiClient.interceptors.request.use(async (config) => {
  if (typeof window !== 'undefined' && process.env.NODE_ENV !== 'development') {
    try {
      const { getAccessToken } = await import('@auth0/nextjs-auth0/client');
      const token = await getAccessToken();
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    } catch {
      // Auth0 session not available — continue without token
    }
  }
  return config;
});

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ error?: string }>) => {
    const message = error.response?.data?.error || error.message || 'An error occurred';

    // Redirect to Auth0 login on 401
    if (error.response?.status === 401 && typeof window !== 'undefined') {
      window.location.href = '/api/auth/login';
      return Promise.reject(new Error('Session expired'));
    }

    // Suppress noise from endpoints the caller handles via try/catch:
    // - 404s from unimplemented endpoints
    // - /metrics 500 (clinical_facts table not yet created)
    const url = error.config?.url ?? '';
    const isSilenced = error.response?.status === 404 || url.includes('/metrics');
    if (!isSilenced) {
      console.error('[Governance API Error]', message);
    }
    return Promise.reject(new Error(message));
  }
);

// ============================================================================
// Queue Operations
// ============================================================================

export const queueApi = {
  /**
   * Get the governance review queue with optional filters
   */
  async getQueue(
    filters?: QueueFilters,
    sort?: SortOptions,
    page = 1,
    pageSize = 20
  ): Promise<PaginatedResponse<QueueItem>> {
    const params = new URLSearchParams();

    if (filters?.status?.length) params.set('status', filters.status.join(','));
    if (filters?.priority?.length) params.set('priority', filters.priority.join(','));
    if (filters?.factType?.length) params.set('factType', filters.factType.join(','));
    if (filters?.assignedTo) params.set('assignedTo', filters.assignedTo);
    if (filters?.slaStatus) params.set('slaStatus', filters.slaStatus);
    if (filters?.hasConflicts !== undefined) params.set('hasConflicts', String(filters.hasConflicts));
    if (filters?.search) params.set('search', filters.search);
    if (sort) {
      params.set('sortBy', sort.field);
      params.set('sortDir', sort.direction);
    }
    // Tell the backend how many items to return (default is 500 which truncates).
    // We paginate client-side, so request enough to cover all items.
    params.set('pageSize', String(pageSize));

    const { data } = await apiClient.get<{ items: QueueItem[]; totalCount: number }>(`/queue?${params}`);
    const allItems = data.items || [];
    const start = (page - 1) * pageSize;
    const paged = allItems.slice(start, start + pageSize);
    return {
      items: paged,
      total: allItems.length,
      page,
      pageSize,
      hasMore: start + pageSize < allItems.length,
    };
  },

  /**
   * Get queue items by priority level
   */
  async getQueueByPriority(priority: string): Promise<QueueItem[]> {
    const { data } = await apiClient.get<{ items: QueueItem[] }>(`/queue/priority/${priority}`);
    return data.items || [];
  },

  /**
   * Get queue items assigned to a specific reviewer
   */
  async getQueueByReviewer(reviewerId: string): Promise<QueueItem[]> {
    const { data } = await apiClient.get<{ items: QueueItem[] }>(`/queue/reviewer/${reviewerId}`);
    return data.items || [];
  },
};

// ============================================================================
// Fact Operations
// ============================================================================

export const factsApi = {
  /**
   * Get all facts with optional filters
   */
  async getAllFacts(
    filters?: { status?: string; factType?: string; search?: string },
    page = 1,
    pageSize = 20
  ): Promise<PaginatedResponse<ClinicalFact>> {
    const params = new URLSearchParams();
    if (filters?.status) params.set('status', filters.status);
    if (filters?.factType) params.set('factType', filters.factType);
    if (filters?.search) params.set('search', filters.search);
    params.set('page', String(page));
    params.set('pageSize', String(pageSize));

    const { data } = await apiClient.get<PaginatedResponse<ClinicalFact>>(`/facts?${params}`);
    return data;
  },

  /**
   * Get a single fact by ID
   */
  async getFact(factId: string): Promise<ClinicalFact> {
    const { data } = await apiClient.get<ClinicalFact>(`/facts/${factId}`);
    if (!data) throw new Error('Fact not found');
    return data;
  },

  /**
   * Get conflicting facts for a given fact
   */
  async getConflicts(factId: string): Promise<ConflictGroup> {
    const { data } = await apiClient.get<ConflictGroup>(`/facts/${factId}/conflicts`);
    if (!data) throw new Error('No conflicts found');
    return data;
  },

  /**
   * Get all conflict groups
   */
  async getAllConflicts(): Promise<ConflictGroup[]> {
    const { data } = await apiClient.get<{ groups: ConflictGroup[] }>('/conflicts');
    return data?.groups || [];
  },

  /**
   * Get audit history for a fact (21 CFR Part 11 compliant audit trail)
   */
  async getHistory(factId: string): Promise<AuditEvent[]> {
    const { data } = await apiClient.get<{ factId: string; items: AuditEvent[]; total: number }>(`/facts/${factId}/history`);
    return data?.items || [];
  },

  /**
   * Get system-wide audit log
   */
  async getAuditLog(
    filters?: { eventType?: string; actorId?: string; fromDate?: string; toDate?: string },
    page = 1,
    pageSize = 50
  ): Promise<PaginatedResponse<AuditEvent>> {
    const params = new URLSearchParams();
    if (filters?.eventType) params.set('eventType', filters.eventType);
    if (filters?.actorId) params.set('actorId', filters.actorId);
    if (filters?.fromDate) params.set('fromDate', filters.fromDate);
    if (filters?.toDate) params.set('toDate', filters.toDate);
    params.set('page', String(page));
    params.set('pageSize', String(pageSize));

    const { data } = await apiClient.get<PaginatedResponse<AuditEvent>>(`/audit?${params}`);
    return data;
  },

  /**
   * Approve a fact
   */
  async approve(request: ReviewRequest): Promise<ReviewDecision> {
    const { data } = await apiClient.post<ReviewDecision>(
      `/facts/${request.factId}/approve`,
      request
    );
    if (!data) throw new Error('Approval failed');
    return data;
  },

  /**
   * Reject a fact
   */
  async reject(request: ReviewRequest): Promise<ReviewDecision> {
    const { data } = await apiClient.post<ReviewDecision>(
      `/facts/${request.factId}/reject`,
      request
    );
    if (!data) throw new Error('Rejection failed');
    return data;
  },

  /**
   * Escalate a fact to CMO
   */
  async escalate(request: ReviewRequest): Promise<ReviewDecision> {
    const { data } = await apiClient.post<ReviewDecision>(
      `/facts/${request.factId}/escalate`,
      request
    );
    if (!data) throw new Error('Escalation failed');
    return data;
  },

  /**
   * Assign a reviewer to a fact
   */
  async assignReviewer(factId: string, reviewerId: string): Promise<void> {
    await apiClient.post(`/facts/${factId}/assign`, { reviewerId });
  },
};

// ============================================================================
// Dashboard & Metrics
// ============================================================================

export const dashboardApi = {
  /**
   * Get governance metrics
   */
  async getMetrics(): Promise<GovernanceMetrics> {
    const { data } = await apiClient.get<GovernanceMetrics>('/metrics');
    return data;
  },

  /**
   * Get full dashboard data
   */
  async getDashboard(): Promise<DashboardData> {
    const { data } = await apiClient.get<DashboardData>('/dashboard');
    return data;
  },
};

// ============================================================================
// Executor Control
// ============================================================================

export const executorApi = {
  /**
   * Get executor status
   */
  async getStatus(): Promise<ExecutorStatus> {
    const { data } = await apiClient.get<ExecutorStatus>('/executor/status');
    return data;
  },

  /**
   * Start the background executor
   */
  async start(): Promise<void> {
    await apiClient.post('/executor/start');
  },

  /**
   * Stop the background executor
   */
  async stop(): Promise<void> {
    await apiClient.post('/executor/stop');
  },

  /**
   * Manually process a single fact
   */
  async processOne(factId: string): Promise<void> {
    await apiClient.post(`/executor/process/${factId}`);
  },
};

// ============================================================================
// Export consolidated API
// ============================================================================

export const governanceApi = {
  queue: queueApi,
  facts: factsApi,
  dashboard: dashboardApi,
  executor: executorApi,
};

export default governanceApi;
