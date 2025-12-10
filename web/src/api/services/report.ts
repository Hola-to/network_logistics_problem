import { typedClient } from "../typedClient";
import { ReportFormat, ReportType } from "@gen/logistics/gateway/v1/gateway_pb";
import type { Graph, FlowResult } from "@gen/logistics/common/v1/common_pb";
import type {
  ReportOptions,
  SolveMetrics,
  ReportChunk,
  FlowReportSource,
} from "@gen/logistics/gateway/v1/gateway_pb";

export { ReportFormat, ReportType };

export const reportService = {
  generateFlowReport: (
    graph: Graph,
    result: FlowResult,
    format: ReportFormat = ReportFormat.PDF,
    options?: Partial<ReportOptions>,
    metrics?: SolveMetrics,
  ) =>
    typedClient.generateReport({
      type: ReportType.FLOW,
      format,
      options: {
        title: "Отчёт по оптимизации потока",
        author: "Logistics Platform",
        language: "ru",
        timezone: "Europe/Moscow",
        includeGraphDetails: true,
        includeEdgeList: true,
        includePathDetails: true,
        includeRecommendations: true,
        currency: "RUB",
        saveToStorage: true,
        ...options,
      } as unknown as ReportOptions,
      source: {
        case: "flowSource" as const,
        value: {
          graph,
          result,
          metrics,
        } as unknown as FlowReportSource,
      },
    }),

  get: (reportId: string) => typedClient.getReport(reportId),

  /**
   * Скачивание отчёта (streaming)
   * Возвращает async generator
   */
  download: async function* (reportId: string): AsyncGenerator<ReportChunk> {
    yield* typedClient.downloadReport(reportId);
  },

  downloadAsBlob: async (reportId: string): Promise<Blob> => {
    return typedClient.downloadReportAsBlob(reportId);
  },

  list: (params?: {
    limit?: number;
    offset?: number;
    type?: ReportType;
    format?: ReportFormat;
  }) =>
    typedClient.listReports({
      limit: params?.limit ?? 20,
      offset: params?.offset ?? 0,
      type: params?.type ?? ReportType.UNSPECIFIED,
      format: params?.format ?? ReportFormat.UNSPECIFIED,
    }),

  delete: (reportId: string) => typedClient.deleteReport(reportId),

  getFormats: () => typedClient.getReportFormats(),
};
