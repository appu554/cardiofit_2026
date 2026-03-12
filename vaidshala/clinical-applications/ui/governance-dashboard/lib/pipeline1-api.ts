// ============================================================================
// Pipeline 1 Review API Client
// Mirrors lib/api.ts pattern — same Auth0 interceptor, separate base URL
// ============================================================================

import axios, { AxiosInstance, AxiosError } from 'axios';
import type {
  ExtractionJob,
  MergedSpan,
  SectionPassage,
  GuidelineTree,
  JobMetrics,
  SpanReviewRequest,
  AddSpanRequest,
  SpanFilters,
  PaginatedSpans,
  JobListResponse,
  PageInfo,
  PageStats,
  PageDecisionRequest,
  RevalidationResult,
  OutputContract,
} from '@/types/pipeline1';

// ============================================================================
// API Client
// ============================================================================

const API_BASE = process.env.NEXT_PUBLIC_PIPELINE1_API_URL || '/api/v2/pipeline1';

const client: AxiosInstance = axios.create({
  baseURL: API_BASE,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
});

// Auth0 token injection (same pattern as lib/api.ts)
// In dev mode, skip token fetch to avoid __txn_ cookie explosion → HTTP 431
client.interceptors.request.use(async (config) => {
  if (typeof window !== 'undefined' && process.env.NODE_ENV !== 'development') {
    try {
      const { getAccessToken } = await import('@auth0/nextjs-auth0/client');
      const token = await getAccessToken();
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    } catch {
      // No session — continue without token
    }
  }
  return config;
});

client.interceptors.response.use(
  (res) => res,
  (error: AxiosError<{ error?: string }>) => {
    const message = error.response?.data?.error || error.message || 'Pipeline 1 API error';
    if (error.response?.status === 401 && typeof window !== 'undefined') {
      window.location.href = '/api/auth/login';
      return Promise.reject(new Error('Session expired'));
    }
    console.error('[Pipeline1 API]', message);
    return Promise.reject(new Error(message));
  }
);

// ============================================================================
// Job Operations
// ============================================================================

export const jobsApi = {
  async list(page = 1, pageSize = 20): Promise<JobListResponse> {
    const { data } = await client.get<JobListResponse>('/jobs', {
      params: { page, pageSize },
    });
    return data;
  },

  async get(jobId: string): Promise<ExtractionJob> {
    const { data } = await client.get<ExtractionJob>(`/jobs/${jobId}`);
    return data;
  },

  async getMetrics(jobId: string): Promise<JobMetrics> {
    const { data } = await client.get<JobMetrics>(`/jobs/${jobId}/metrics`);
    return data;
  },

  async complete(jobId: string, req: { reviewerId: string; note?: string }): Promise<void> {
    await client.post(`/jobs/${jobId}/complete`, req);
  },
};

// ============================================================================
// Span Operations
// ============================================================================

export const spansApi = {
  async list(
    jobId: string,
    filters?: SpanFilters,
    page = 1,
    pageSize = 50
  ): Promise<PaginatedSpans> {
    const params = new URLSearchParams();
    params.set('page', String(page));
    params.set('pageSize', String(pageSize));

    if (filters?.status) params.set('status', filters.status);
    if (filters?.sectionId) params.set('sectionId', filters.sectionId);
    if (filters?.pageNumber != null) params.set('pageNumber', String(filters.pageNumber));
    if (filters?.minConfidence != null) params.set('minConfidence', String(filters.minConfidence));
    if (filters?.maxConfidence != null) params.set('maxConfidence', String(filters.maxConfidence));
    if (filters?.hasDisagreement != null) params.set('hasDisagreement', String(filters.hasDisagreement));
    if (filters?.search) params.set('search', filters.search);
    if (filters?.tier != null) params.set('tier', String(filters.tier));

    const { data } = await client.get<PaginatedSpans>(`/jobs/${jobId}/spans?${params}`);
    return data;
  },

  async get(jobId: string, spanId: string): Promise<MergedSpan> {
    const { data } = await client.get<MergedSpan>(`/jobs/${jobId}/spans/${spanId}`);
    return data;
  },

  async confirm(jobId: string, spanId: string, req: SpanReviewRequest): Promise<void> {
    await client.post(`/jobs/${jobId}/spans/${spanId}/confirm`, req);
  },

  async reject(jobId: string, spanId: string, req: SpanReviewRequest): Promise<void> {
    await client.post(`/jobs/${jobId}/spans/${spanId}/reject`, req);
  },

  async edit(jobId: string, spanId: string, req: SpanReviewRequest): Promise<void> {
    await client.post(`/jobs/${jobId}/spans/${spanId}/edit`, req);
  },

  async add(jobId: string, req: AddSpanRequest): Promise<void> {
    await client.post(`/jobs/${jobId}/spans/add`, req);
  },
};

// ============================================================================
// Context Operations (Passages, Tree, Text)
// ============================================================================

export const contextApi = {
  async getPassages(jobId: string): Promise<SectionPassage[]> {
    const { data } = await client.get<{ items: SectionPassage[] }>(`/jobs/${jobId}/passages`);
    return data.items || [];
  },

  async getTree(jobId: string): Promise<GuidelineTree> {
    const { data } = await client.get<GuidelineTree>(`/jobs/${jobId}/tree`);
    return data;
  },

  async getText(jobId: string): Promise<string> {
    const { data } = await client.get<{ text: string }>(`/jobs/${jobId}/text`);
    return data.text || '';
  },

  /** Direct URL for iframe src — returns raw text/html from the pipeline highlight output */
  getHighlightHtmlUrl(jobId: string): string {
    return `${API_BASE}/jobs/${jobId}/highlight-html`;
  },

  /** Direct URL for iframe src — streams the original source PDF */
  getSourcePdfUrl(jobId: string): string {
    return `${API_BASE}/jobs/${jobId}/source-pdf`;
  },
};

// ============================================================================
// Page Decision Operations
// ============================================================================

export const pagesApi = {
  async list(jobId: string): Promise<PageInfo[]> {
    const { data } = await client.get<{ items: PageInfo[] }>(`/jobs/${jobId}/pages`);
    return data.items || [];
  },

  async decide(jobId: string, pageNumber: number, req: PageDecisionRequest): Promise<void> {
    await client.post(`/jobs/${jobId}/pages/${pageNumber}/decide`, req);
  },

  async getStats(jobId: string): Promise<PageStats> {
    const { data } = await client.get<PageStats>(`/jobs/${jobId}/pages/stats`);
    return data;
  },
};

// ============================================================================
// Review Task Queue Operations
// ============================================================================

export const reviewTasksApi = {
  async list(jobId: string): Promise<import('@/types/pipeline1').ReviewTask[]> {
    const { data } = await client.get<{ items: import('@/types/pipeline1').ReviewTask[] }>(
      `/jobs/${jobId}/review-tasks`
    );
    return data.items || [];
  },
};

// ============================================================================
// Patched Passages Operations (L3 consumption)
// ============================================================================

export const patchedPassagesApi = {
  async list(jobId: string): Promise<import('@/types/pipeline1').PatchedPassage[]> {
    const { data } = await client.get<{ items: import('@/types/pipeline1').PatchedPassage[] }>(
      `/jobs/${jobId}/passages/patched`
    );
    return data.items || [];
  },
};

// ============================================================================
// Revalidation Operations (Phase 4 — CoverageGuard Delta Check)
// ============================================================================

export const revalidationApi = {
  async run(jobId: string): Promise<RevalidationResult> {
    const { data } = await client.post<RevalidationResult>(`/jobs/${jobId}/revalidate`);
    return data;
  },

  async getHistory(jobId: string): Promise<RevalidationResult[]> {
    const { data } = await client.get<{ items: RevalidationResult[] }>(
      `/jobs/${jobId}/revalidation-history`
    );
    return data.items || [];
  },
};

// ============================================================================
// Output Contract Operations (Phase 5 — Pipeline 1→2 Handoff)
// ============================================================================

export const outputContractApi = {
  async assemble(jobId: string, reviewerId: string): Promise<OutputContract> {
    const { data } = await client.post<OutputContract>(
      `/jobs/${jobId}/output-contract`,
      { reviewerId }
    );
    return data;
  },

  async preview(jobId: string): Promise<OutputContract> {
    const { data } = await client.get<OutputContract>(`/jobs/${jobId}/output-contract/preview`);
    return data;
  },
};

// ============================================================================
// Consolidated Export
// ============================================================================

export const pipeline1Api = {
  jobs: jobsApi,
  spans: spansApi,
  context: contextApi,
  pages: pagesApi,
  reviewTasks: reviewTasksApi,
  patchedPassages: patchedPassagesApi,
  revalidation: revalidationApi,
  outputContract: outputContractApi,
};

export default pipeline1Api;
