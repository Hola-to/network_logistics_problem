// api/typedClient.ts
import { client } from "./client";
import type {
  // Health & Info
  HealthResponse,
  ReadinessResponse,
  InfoResponse,
  AlgorithmsResponse,

  // Auth
  AuthResponse,
  UserProfile,
  ValidateTokenResponse,
  RegisterRequest,
  LoginRequest,
  RefreshTokenRequest,
  ValidateTokenRequest,

  // Optimization
  CalculateLogisticsRequest,
  CalculateLogisticsResponse,
  SolveGraphRequest,
  SolveGraphResponse,
  SolveProgressEvent,
  BatchSolveRequest,
  BatchSolveResponse,

  // Validation
  ValidateGraphRequest,
  ValidateGraphResponse,
  ValidateForAlgorithmRequest,
  ValidateForAlgorithmResponse,

  // Analytics
  AnalyzeGraphRequest,
  AnalyzeGraphResponse,
  CalculateCostRequest,
  CalculateCostResponse,
  BottlenecksRequest,
  BottlenecksResponse,
  CompareScenariosRequest,
  CompareScenariosResponse,

  // Simulation
  WhatIfRequest,
  WhatIfResponse,
  MonteCarloRequest,
  MonteCarloResponse,
  MonteCarloProgressEvent,
  SensitivityRequest,
  SensitivityResponse,
  ResilienceRequest,
  ResilienceResponse,
  FailureSimulationRequest,
  FailureSimulationResponse,
  CriticalElementsRequest,
  CriticalElementsResponse,
  SimulationRecord,
  ListSimulationsRequest,
  ListSimulationsResponse,

  // History
  SaveCalculationRequest,
  SaveCalculationResponse,
  CalculationRecord,
  ListCalculationsRequest,
  ListCalculationsResponse,
  GetStatisticsRequest,
  StatisticsResponse,

  // Reports
  GenerateReportRequest,
  GenerateReportResponse,
  ReportRecord,
  ReportChunk,
  ListReportsRequest,
  ListReportsResponse,
  ReportFormatsResponse,

  // Audit
  GetAuditLogsRequest,
  AuditLogsResponse,
  GetUserActivityRequest,
  UserActivityResponse,
  GetAuditStatsRequest,
  AuditStatsResponse,
} from "@gen/logistics/gateway/v1/gateway_pb";

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Приводит Promise к нужному типу
 */
function typed<T>(promise: Promise<unknown>): Promise<T> {
  return promise as Promise<T>;
}

/**
 * Приводит AsyncIterable к типизированному AsyncGenerator
 */
async function* typedStream<T>(
  stream: AsyncIterable<unknown>,
): AsyncGenerator<T, void, unknown> {
  for await (const item of stream) {
    yield item as T;
  }
}

// ============================================================================
// Typed Client Interface
// ============================================================================

export interface TypedGatewayClient {
  // Health & Info
  health(): Promise<HealthResponse>;
  readinessCheck(): Promise<ReadinessResponse>;
  info(): Promise<InfoResponse>;
  getAlgorithms(): Promise<AlgorithmsResponse>;

  // Auth
  register(data: Partial<RegisterRequest>): Promise<AuthResponse>;
  login(data: Partial<LoginRequest>): Promise<AuthResponse>;
  refreshToken(data: Partial<RefreshTokenRequest>): Promise<AuthResponse>;
  logout(): Promise<void>;
  getProfile(): Promise<UserProfile>;
  validateToken(
    data: Partial<ValidateTokenRequest>,
  ): Promise<ValidateTokenResponse>;

  // Optimization
  calculateLogistics(
    data: Partial<CalculateLogisticsRequest>,
  ): Promise<CalculateLogisticsResponse>;
  solveGraph(data: Partial<SolveGraphRequest>): Promise<SolveGraphResponse>;
  solveGraphStream(
    data: Partial<SolveGraphRequest>,
  ): AsyncGenerator<SolveProgressEvent>;
  batchSolve(data: Partial<BatchSolveRequest>): Promise<BatchSolveResponse>;

  // Validation
  validateGraph(
    data: Partial<ValidateGraphRequest>,
  ): Promise<ValidateGraphResponse>;
  validateForAlgorithm(
    data: Partial<ValidateForAlgorithmRequest>,
  ): Promise<ValidateForAlgorithmResponse>;

  // Analytics
  analyzeGraph(
    data: Partial<AnalyzeGraphRequest>,
  ): Promise<AnalyzeGraphResponse>;
  calculateCost(
    data: Partial<CalculateCostRequest>,
  ): Promise<CalculateCostResponse>;
  getBottlenecks(
    data: Partial<BottlenecksRequest>,
  ): Promise<BottlenecksResponse>;
  compareScenarios(
    data: Partial<CompareScenariosRequest>,
  ): Promise<CompareScenariosResponse>;

  // Simulation
  runWhatIf(data: Partial<WhatIfRequest>): Promise<WhatIfResponse>;
  runMonteCarlo(data: Partial<MonteCarloRequest>): Promise<MonteCarloResponse>;
  runMonteCarloStream(
    data: Partial<MonteCarloRequest>,
  ): AsyncGenerator<MonteCarloProgressEvent>;
  analyzeSensitivity(
    data: Partial<SensitivityRequest>,
  ): Promise<SensitivityResponse>;
  analyzeResilience(
    data: Partial<ResilienceRequest>,
  ): Promise<ResilienceResponse>;
  simulateFailures(
    data: Partial<FailureSimulationRequest>,
  ): Promise<FailureSimulationResponse>;
  findCriticalElements(
    data: Partial<CriticalElementsRequest>,
  ): Promise<CriticalElementsResponse>;
  getSimulation(simulationId: string): Promise<SimulationRecord>;
  listSimulations(
    data?: Partial<ListSimulationsRequest>,
  ): Promise<ListSimulationsResponse>;
  deleteSimulation(simulationId: string): Promise<void>;

  // History
  saveCalculation(
    data: Partial<SaveCalculationRequest>,
  ): Promise<SaveCalculationResponse>;
  getCalculation(calculationId: string): Promise<CalculationRecord>;
  listCalculations(
    data?: Partial<ListCalculationsRequest>,
  ): Promise<ListCalculationsResponse>;
  deleteCalculation(calculationId: string): Promise<void>;
  getStatistics(
    data?: Partial<GetStatisticsRequest>,
  ): Promise<StatisticsResponse>;

  // Reports
  generateReport(
    data: Partial<GenerateReportRequest>,
  ): Promise<GenerateReportResponse>;
  getReport(reportId: string): Promise<ReportRecord>;
  downloadReport(reportId: string): AsyncGenerator<ReportChunk>;
  downloadReportAsBlob(reportId: string): Promise<Blob>;
  listReports(data?: Partial<ListReportsRequest>): Promise<ListReportsResponse>;
  deleteReport(reportId: string): Promise<void>;
  getReportFormats(): Promise<ReportFormatsResponse>;

  // Audit
  getAuditLogs(data?: Partial<GetAuditLogsRequest>): Promise<AuditLogsResponse>;
  getUserActivity(
    data: Partial<GetUserActivityRequest>,
  ): Promise<UserActivityResponse>;
  getAuditStats(
    data?: Partial<GetAuditStatsRequest>,
  ): Promise<AuditStatsResponse>;
}

// ============================================================================
// Typed Client Implementation
// ============================================================================

export const typedClient: TypedGatewayClient = {
  // ==========================================================================
  // Health & Info
  // ==========================================================================

  health: () => typed<HealthResponse>(client.health({})),

  readinessCheck: () => typed<ReadinessResponse>(client.readinessCheck({})),

  info: () => typed<InfoResponse>(client.info({})),

  getAlgorithms: () => typed<AlgorithmsResponse>(client.getAlgorithms({})),

  // ==========================================================================
  // Auth
  // ==========================================================================

  register: (data) => typed<AuthResponse>(client.register(data)),

  login: (data) => typed<AuthResponse>(client.login(data)),

  refreshToken: (data) => typed<AuthResponse>(client.refreshToken(data)),

  logout: async () => {
    await client.logout({});
  },

  getProfile: () => typed<UserProfile>(client.getProfile({})),

  validateToken: (data) =>
    typed<ValidateTokenResponse>(client.validateToken(data)),

  // ==========================================================================
  // Optimization
  // ==========================================================================

  calculateLogistics: (data) =>
    typed<CalculateLogisticsResponse>(client.calculateLogistics(data)),

  solveGraph: (data) => typed<SolveGraphResponse>(client.solveGraph(data)),

  solveGraphStream: async function* (data) {
    const stream = client.solveGraphStream(data);
    yield* typedStream<SolveProgressEvent>(stream);
  },

  batchSolve: (data) => typed<BatchSolveResponse>(client.batchSolve(data)),

  // ==========================================================================
  // Validation
  // ==========================================================================

  validateGraph: (data) =>
    typed<ValidateGraphResponse>(client.validateGraph(data)),

  validateForAlgorithm: (data) =>
    typed<ValidateForAlgorithmResponse>(client.validateForAlgorithm(data)),

  // ==========================================================================
  // Analytics
  // ==========================================================================

  analyzeGraph: (data) =>
    typed<AnalyzeGraphResponse>(client.analyzeGraph(data)),

  calculateCost: (data) =>
    typed<CalculateCostResponse>(client.calculateCost(data)),

  getBottlenecks: (data) =>
    typed<BottlenecksResponse>(client.getBottlenecks(data)),

  compareScenarios: (data) =>
    typed<CompareScenariosResponse>(client.compareScenarios(data)),

  // ==========================================================================
  // Simulation
  // ==========================================================================

  runWhatIf: (data) => typed<WhatIfResponse>(client.runWhatIf(data)),

  runMonteCarlo: (data) =>
    typed<MonteCarloResponse>(client.runMonteCarlo(data)),

  runMonteCarloStream: async function* (data) {
    const stream = client.runMonteCarloStream(data);
    yield* typedStream<MonteCarloProgressEvent>(stream);
  },

  analyzeSensitivity: (data) =>
    typed<SensitivityResponse>(client.analyzeSensitivity(data)),

  analyzeResilience: (data) =>
    typed<ResilienceResponse>(client.analyzeResilience(data)),

  simulateFailures: (data) =>
    typed<FailureSimulationResponse>(client.simulateFailures(data)),

  findCriticalElements: (data) =>
    typed<CriticalElementsResponse>(client.findCriticalElements(data)),

  getSimulation: (simulationId) =>
    typed<SimulationRecord>(client.getSimulation({ simulationId })),

  listSimulations: (data = {}) =>
    typed<ListSimulationsResponse>(client.listSimulations(data)),

  deleteSimulation: async (simulationId) => {
    await client.deleteSimulation({ simulationId });
  },

  // ==========================================================================
  // History
  // ==========================================================================

  saveCalculation: (data) =>
    typed<SaveCalculationResponse>(client.saveCalculation(data)),

  getCalculation: (calculationId) =>
    typed<CalculationRecord>(client.getCalculation({ calculationId })),

  listCalculations: (data = {}) =>
    typed<ListCalculationsResponse>(client.listCalculations(data)),

  deleteCalculation: async (calculationId) => {
    await client.deleteCalculation({ calculationId });
  },

  getStatistics: (data = {}) =>
    typed<StatisticsResponse>(client.getStatistics(data)),

  // ==========================================================================
  // Reports
  // ==========================================================================

  generateReport: (data) =>
    typed<GenerateReportResponse>(client.generateReport(data)),

  getReport: (reportId) => typed<ReportRecord>(client.getReport({ reportId })),

  downloadReport: async function* (reportId) {
    const stream = client.downloadReport({ reportId });
    yield* typedStream<ReportChunk>(stream);
  },

  downloadReportAsBlob: async (reportId) => {
    const chunks: Uint8Array[] = [];
    const stream = client.downloadReport({ reportId });

    for await (const chunk of stream) {
      const typedChunk = chunk as unknown as ReportChunk;
      chunks.push(typedChunk.data);
    }

    return new Blob(chunks as BlobPart[]);
  },

  listReports: (data = {}) =>
    typed<ListReportsResponse>(client.listReports(data)),

  deleteReport: async (reportId) => {
    await client.deleteReport({ reportId });
  },

  getReportFormats: () =>
    typed<ReportFormatsResponse>(client.getReportFormats({})),

  // ==========================================================================
  // Audit
  // ==========================================================================

  getAuditLogs: (data = {}) =>
    typed<AuditLogsResponse>(client.getAuditLogs(data)),

  getUserActivity: (data) =>
    typed<UserActivityResponse>(client.getUserActivity(data)),

  getAuditStats: (data = {}) =>
    typed<AuditStatsResponse>(client.getAuditStats(data)),
};

// ============================================================================
// Default Export
// ============================================================================

export default typedClient;
