import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { format } from "date-fns";
import { ru } from "date-fns/locale";
import {
  DocumentArrowDownIcon,
  TrashIcon,
  DocumentTextIcon,
  TableCellsIcon,
  CodeBracketIcon,
} from "@heroicons/react/24/outline";
import Card, { CardHeader } from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";
import Spinner from "@/components/ui/Spinner";
import { useGraphStore } from "@/stores/graphStore";
import { reportService, ReportFormat } from "@/api/services/report";
import type {
  ReportInfo,
  GenerateReportResponse,
} from "@gen/logistics/gateway/v1/gateway_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import clsx from "clsx";

const FORMAT_CONFIG: Record<
  number,
  { icon: typeof DocumentTextIcon; label: string; ext: string; mime: string }
> = {
  [ReportFormat.MARKDOWN]: {
    icon: DocumentTextIcon,
    label: "Markdown",
    ext: "md",
    mime: "text/markdown",
  },
  [ReportFormat.CSV]: {
    icon: TableCellsIcon,
    label: "CSV",
    ext: "csv",
    mime: "text/csv",
  },
  [ReportFormat.EXCEL]: {
    icon: TableCellsIcon,
    label: "Excel",
    ext: "xlsx",
    mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  },
  [ReportFormat.PDF]: {
    icon: DocumentTextIcon,
    label: "PDF",
    ext: "pdf",
    mime: "application/pdf",
  },
  [ReportFormat.HTML]: {
    icon: CodeBracketIcon,
    label: "HTML",
    ext: "html",
    mime: "text/html",
  },
  [ReportFormat.JSON]: {
    icon: CodeBracketIcon,
    label: "JSON",
    ext: "json",
    mime: "application/json",
  },
};

const AVAILABLE_FORMATS = [
  ReportFormat.PDF,
  ReportFormat.EXCEL,
  ReportFormat.CSV,
  ReportFormat.HTML,
  ReportFormat.MARKDOWN,
  ReportFormat.JSON,
];

export default function Reports() {
  const queryClient = useQueryClient();
  const { solvedGraph, flowResult, metrics } = useGraphStore();
  const [selectedFormat, setSelectedFormat] = useState<ReportFormat>(
    ReportFormat.PDF,
  );
  const [reportTitle, setReportTitle] = useState("Отчёт по оптимизации");

  const hasSolution = !!flowResult;

  const reportsQuery = useQuery({
    queryKey: ["reports"],
    queryFn: () => reportService.list({ limit: 20 }),
  });

  const generateMutation = useMutation({
    mutationFn: async () => {
      if (!flowResult || !solvedGraph) {
        throw new Error("Нет данных для отчёта");
      }

      return reportService.generateFlowReport(
        solvedGraph,
        flowResult,
        selectedFormat,
        {
          title: reportTitle,
          author: "Logistics Platform",
          includeGraphDetails: true,
          includeEdgeList: true,
          includePathDetails: true,
          includeRecommendations: true,
          currency: "RUB",
          saveToStorage: true,
        },
        metrics ?? undefined,
      );
    },
    onSuccess: (response: GenerateReportResponse) => {
      if (response.success && response.content) {
        const config = FORMAT_CONFIG[selectedFormat];
        const blob = new Blob([response.content as BlobPart], {
          type: config?.mime ?? "application/octet-stream",
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download =
          response.report?.filename ?? `report.${config?.ext ?? "bin"}`;
        a.click();
        URL.revokeObjectURL(url);

        toast.success("Отчёт сгенерирован");
        queryClient.invalidateQueries({ queryKey: ["reports"] });
      } else {
        toast.error(response.errorMessage || "Ошибка генерации");
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const handleDownload = async (report: ReportInfo) => {
    try {
      const blob = await reportService.downloadAsBlob(report.reportId);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = report.filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      toast.error("Ошибка скачивания");
    }
  };

  const deleteMutation = useMutation({
    mutationFn: (id: string) => reportService.delete(id),
    onSuccess: () => {
      toast.success("Отчёт удалён");
      queryClient.invalidateQueries({ queryKey: ["reports"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Отчёты</h1>

      {/* Generate new report */}
      <Card>
        <CardHeader title="Создать отчёт" />

        {!hasSolution ? (
          <div className="text-center py-8">
            <p className="text-gray-500">
              Сначала выполните оптимизацию в редакторе сети
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            <Input
              label="Название отчёта"
              value={reportTitle}
              onChange={(e) => setReportTitle(e.target.value)}
            />

            <div>
              <label className="label">Формат</label>
              <div className="grid grid-cols-3 md:grid-cols-6 gap-2">
                {AVAILABLE_FORMATS.map((format) => {
                  const config = FORMAT_CONFIG[format];
                  const Icon = config?.icon ?? DocumentTextIcon;
                  return (
                    <button
                      key={format}
                      onClick={() => setSelectedFormat(format)}
                      className={clsx(
                        "flex flex-col items-center p-3 rounded-lg border-2 transition-colors",
                        selectedFormat === format
                          ? "border-primary-500 bg-primary-50"
                          : "border-gray-200 hover:border-gray-300",
                      )}
                    >
                      <Icon className="w-6 h-6 mb-1" />
                      <span className="text-sm">{config?.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>

            <Button
              onClick={() => generateMutation.mutate()}
              loading={generateMutation.isPending}
            >
              Сгенерировать отчёт
            </Button>
          </div>
        )}
      </Card>

      {/* Existing reports */}
      <Card>
        <CardHeader title="Сохранённые отчёты" />

        {reportsQuery.isLoading ? (
          <div className="flex justify-center py-8">
            <Spinner />
          </div>
        ) : !reportsQuery.data?.reports?.length ? (
          <p className="text-gray-500 text-center py-8">
            Нет сохранённых отчётов
          </p>
        ) : (
          <div className="space-y-3">
            {reportsQuery.data.reports.map((report: ReportInfo) => {
              const config = FORMAT_CONFIG[report.format];
              const Icon = config?.icon ?? DocumentTextIcon;
              return (
                <div
                  key={report.reportId}
                  className="flex items-center justify-between p-4 bg-gray-50 rounded-lg"
                >
                  <div className="flex items-center gap-3">
                    <Icon className="w-8 h-8 text-gray-400" />
                    <div>
                      <p className="font-medium">{report.title}</p>
                      <p className="text-sm text-gray-500">
                        {report.generatedAt
                          ? format(
                              timestampDate(report.generatedAt),
                              "dd.MM.yyyy HH:mm",
                              { locale: ru },
                            )
                          : "—"}{" "}
                        • {(Number(report.sizeBytes) / 1024).toFixed(1)} KB
                      </p>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDownload(report)}
                    >
                      <DocumentArrowDownIcon className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => deleteMutation.mutate(report.reportId)}
                      loading={deleteMutation.isPending}
                    >
                      <TrashIcon className="w-4 h-4 text-red-500" />
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </Card>
    </div>
  );
}
