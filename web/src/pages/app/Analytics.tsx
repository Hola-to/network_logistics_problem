import { useState, useMemo } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import Card, { CardHeader } from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Spinner from "@/components/ui/Spinner";
import { useGraphStore } from "@/stores/graphStore";
import { analyticsService } from "@/api/services";
import {
  UtilizationBarChart,
  CostPieChart,
} from "@/components/visual/FlowChart";
import {
  BottleneckSeverity,
  type Bottleneck,
  type Recommendation,
} from "@gen/logistics/gateway/v1/gateway_pb";
import clsx from "clsx";

const SEVERITY_COLORS: Record<number, string> = {
  [BottleneckSeverity.LOW]: "bg-yellow-100 text-yellow-800 border-yellow-200",
  [BottleneckSeverity.MEDIUM]:
    "bg-orange-100 text-orange-800 border-orange-200",
  [BottleneckSeverity.HIGH]: "bg-red-100 text-red-800 border-red-200",
  [BottleneckSeverity.CRITICAL]: "bg-red-200 text-red-900 border-red-300",
};

// Генерируем стабильный ключ для графа (без BigInt)
const getGraphKey = (
  nodes: { id: bigint }[],
  edges: { from: bigint; to: bigint }[],
): string => {
  const nodeIds = nodes
    .map((n) => String(n.id))
    .sort()
    .join(",");
  const edgeIds = edges
    .map((e) => `${e.from}-${e.to}`)
    .sort()
    .join(",");
  return `${nodeIds}|${edgeIds}`;
};

export default function Analytics() {
  const { getGraph, solvedGraph, flowResult, nodes, edges } = useGraphStore();
  const [currency, setCurrency] = useState("RUB");

  const hasGraph = nodes.length > 0;
  const hasSolution = !!flowResult;

  // Стабильный ключ для кэширования (без BigInt)
  const graphKey = useMemo(() => getGraphKey(nodes, edges), [nodes, edges]);

  // Full analysis query
  const analysisQuery = useQuery({
    queryKey: ["analysis", graphKey, hasSolution], // ← Используем строковый ключ
    queryFn: () => {
      const graphToAnalyze = solvedGraph ?? getGraph();
      return analyticsService.analyzeGraph({
        graph: graphToAnalyze,
        options: {
          analyzeCosts: true,
          findBottlenecks: true,
          calculateStatistics: true,
          suggestImprovements: true,
          bottleneckThreshold: 0.9,
        },
      });
    },
    enabled: hasGraph && hasSolution,
  });

  // Cost calculation mutation
  const costMutation = useMutation({
    mutationFn: () => {
      const graphToAnalyze = solvedGraph ?? getGraph();
      return analyticsService.calculateCost({
        graph: graphToAnalyze,
        options: {
          currency,
          includeFixedCosts: true,
        },
      });
    },
    onSuccess: () => toast.success("Стоимость рассчитана"),
    onError: (error: Error) => toast.error(error.message),
  });

  const analysis = analysisQuery.data;

  // Prepare chart data - конвертируем BigInt в строки
  const utilizationData = useMemo(() => {
    if (!solvedGraph?.edges) return [];
    return solvedGraph.edges.map((e) => ({
      name: `${String(e.from)}→${String(e.to)}`,
      utilization:
        e.capacity > 0 ? ((e.currentFlow ?? 0) / e.capacity) * 100 : 0,
    }));
  }, [solvedGraph]);

  const costData = useMemo(() => {
    if (!analysis?.cost?.breakdown) return [];
    return [
      {
        name: "Транспорт",
        value: analysis.cost.breakdown.transportCost ?? 0,
      },
      {
        name: "Фикс. затраты",
        value: analysis.cost.breakdown.fixedCost ?? 0,
      },
      { name: "Обработка", value: analysis.cost.breakdown.handlingCost ?? 0 },
    ].filter((d) => d.value > 0);
  }, [analysis]);

  if (analysisQuery.isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Аналитика</h1>
        {hasSolution && (
          <Button
            variant="secondary"
            onClick={() => analysisQuery.refetch()}
            loading={analysisQuery.isFetching}
          >
            Обновить
          </Button>
        )}
      </div>

      {!hasGraph && (
        <Card className="bg-yellow-50 border-yellow-200">
          <p className="text-yellow-800">Создайте граф в редакторе сети</p>
        </Card>
      )}

      {hasGraph && !hasSolution && (
        <Card className="bg-blue-50 border-blue-200">
          <p className="text-blue-800">
            Запустите оптимизацию для получения аналитики
          </p>
        </Card>
      )}

      {hasSolution && (
        <>
          {/* Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <Card>
              <p className="text-sm text-gray-500">Максимальный поток</p>
              <p className="text-3xl font-bold text-primary-600">
                {flowResult?.maxFlow}
              </p>
            </Card>
            <Card>
              <p className="text-sm text-gray-500">Общая стоимость</p>
              <p className="text-3xl font-bold text-gray-900">
                ₽{(flowResult?.totalCost ?? 0).toFixed(0)}
              </p>
            </Card>
            <Card>
              <p className="text-sm text-gray-500">Эффективность</p>
              <p className="text-3xl font-bold text-green-600">
                {analysis?.efficiency?.grade ?? "—"}
              </p>
              <p className="text-sm text-gray-500">
                {((analysis?.efficiency?.overallEfficiency ?? 0) * 100).toFixed(
                  1,
                )}
                %
              </p>
            </Card>
            <Card>
              <p className="text-sm text-gray-500">Узких мест</p>
              <p className="text-3xl font-bold text-orange-600">
                {analysis?.bottlenecks?.totalBottlenecks ?? 0}
              </p>
            </Card>
          </div>

          {/* Charts */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Utilization Chart */}
            <Card>
              <CardHeader title="Загрузка рёбер" />
              {utilizationData.length > 0 ? (
                <UtilizationBarChart
                  data={utilizationData.slice(0, 10)}
                  height={300}
                />
              ) : (
                <p className="text-gray-500">Нет данных</p>
              )}
            </Card>

            {/* Cost Chart */}
            <Card>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold">Структура затрат</h3>
                <select
                  value={currency}
                  onChange={(e) => setCurrency(e.target.value)}
                  className="input w-24"
                >
                  <option value="RUB">₽ RUB</option>
                  <option value="USD">$ USD</option>
                  <option value="EUR">€ EUR</option>
                </select>
              </div>
              {costData.length > 0 ? (
                <CostPieChart data={costData} height={300} />
              ) : (
                <div className="text-center py-8">
                  <p className="text-gray-500 mb-4">
                    Расчёт стоимости не выполнен
                  </p>
                  <Button
                    variant="secondary"
                    onClick={() => costMutation.mutate()}
                    loading={costMutation.isPending}
                  >
                    Рассчитать
                  </Button>
                </div>
              )}
            </Card>
          </div>

          {/* Bottlenecks */}
          {analysis?.bottlenecks &&
            analysis.bottlenecks.bottlenecks.length > 0 && (
              <Card>
                <CardHeader title="Узкие места" />
                <div className="space-y-3">
                  {analysis.bottlenecks.bottlenecks.map(
                    (b: Bottleneck, i: number) => (
                      <div
                        key={i}
                        className={clsx(
                          "p-4 rounded-lg border",
                          SEVERITY_COLORS[b.severity] ??
                            "bg-gray-50 border-gray-200",
                        )}
                      >
                        <div className="flex items-center justify-between">
                          <div>
                            <span className="font-medium">
                              Ребро {String(b.edge?.from)} →{" "}
                              {String(b.edge?.to)}
                            </span>
                            <span className="ml-2 text-sm">
                              Загрузка: {(b.utilization * 100).toFixed(1)}%
                            </span>
                          </div>
                          <div className="text-right">
                            <p className="text-sm">
                              Impact Score:{" "}
                              <strong>{b.impactScore.toFixed(2)}</strong>
                            </p>
                          </div>
                        </div>
                      </div>
                    ),
                  )}
                </div>
              </Card>
            )}

          {/* Recommendations */}
          {analysis?.bottlenecks &&
            analysis.bottlenecks.recommendations.length > 0 && (
              <Card>
                <CardHeader title="Рекомендации" />
                <div className="space-y-3">
                  {analysis.bottlenecks.recommendations.map(
                    (r: Recommendation, i: number) => (
                      <div
                        key={i}
                        className="p-4 bg-blue-50 border border-blue-200 rounded-lg"
                      >
                        <div className="flex items-start justify-between">
                          <div>
                            <span className="inline-block px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded mb-2">
                              {r.type}
                            </span>
                            <p className="text-gray-700">{r.description}</p>
                            {r.affectedEdge && (
                              <p className="text-sm text-gray-500 mt-1">
                                Ребро: {String(r.affectedEdge.from)} →{" "}
                                {String(r.affectedEdge.to)}
                              </p>
                            )}
                          </div>
                          <div className="text-right text-sm">
                            <p className="text-green-600">
                              +{r.estimatedImprovement.toFixed(1)}% потока
                            </p>
                            <p className="text-gray-500">
                              ₽{r.estimatedCost.toFixed(0)}
                            </p>
                          </div>
                        </div>
                      </div>
                    ),
                  )}
                </div>
              </Card>
            )}

          {/* Graph Statistics */}
          {analysis?.graphStats && (
            <Card>
              <CardHeader title="Статистика графа" />
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div>
                  <p className="text-sm text-gray-500">Узлов</p>
                  <p className="text-xl font-bold">
                    {String(analysis.graphStats.nodeCount)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Рёбер</p>
                  <p className="text-xl font-bold">
                    {String(analysis.graphStats.edgeCount)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Складов</p>
                  <p className="text-xl font-bold">
                    {String(analysis.graphStats.warehouseCount)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Точек доставки</p>
                  <p className="text-xl font-bold">
                    {String(analysis.graphStats.deliveryPointCount)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Общая capacity</p>
                  <p className="text-xl font-bold">
                    {analysis.graphStats.totalCapacity}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Связность</p>
                  <p className="text-xl font-bold">
                    {analysis.graphStats.isConnected ? "✓ Да" : "✗ Нет"}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Плотность</p>
                  <p className="text-xl font-bold">
                    {(analysis.graphStats.density * 100).toFixed(1)}%
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Средняя длина ребра</p>
                  <p className="text-xl font-bold">
                    {analysis.graphStats.averageEdgeLength.toFixed(1)}
                  </p>
                </div>
              </div>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
