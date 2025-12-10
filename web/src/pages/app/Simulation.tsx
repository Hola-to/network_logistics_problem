import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Tab } from "@headlessui/react";
import clsx from "clsx";
import Card, { CardHeader } from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";
import Select from "@/components/ui/Select";
import { useGraphStore } from "@/stores/graphStore";
import { simulationService } from "@/api/services";
import { SensitivityLineChart } from "@/components/visual/FlowChart";
import { Algorithm } from "@gen/logistics/common/v1/common_pb";
import {
  ModificationType,
  ModificationTarget,
  DistributionType,
  ImpactLevel,
} from "@gen/logistics/gateway/v1/gateway_pb";
import type {
  Modification,
  WhatIfResponse,
  MonteCarloResponse,
  SensitivityResponse,
  UncertaintySpec,
  SensitivityParameter,
} from "@gen/logistics/gateway/v1/gateway_pb";

const IMPACT_COLORS: Record<number, string> = {
  [ImpactLevel.NONE]: "bg-gray-100 text-gray-800",
  [ImpactLevel.LOW]: "bg-green-100 text-green-800",
  [ImpactLevel.MEDIUM]: "bg-yellow-100 text-yellow-800",
  [ImpactLevel.HIGH]: "bg-orange-100 text-orange-800",
  [ImpactLevel.CRITICAL]: "bg-red-100 text-red-800",
};

export default function Simulation() {
  const { getGraph, algorithm, nodes, edges } = useGraphStore();
  const [activeTab, setActiveTab] = useState(0);

  // What-If state
  const [modifications, setModifications] = useState<Modification[]>([]);
  const [whatIfResult, setWhatIfResult] = useState<WhatIfResponse | null>(null);

  // Monte Carlo state
  const [mcIterations, setMcIterations] = useState(1000);
  const [mcConfidence, setMcConfidence] = useState(0.95);
  const [mcResult, setMcResult] = useState<MonteCarloResponse | null>(null);

  // Sensitivity state
  const [sensitivityResult, setSensitivityResult] =
    useState<SensitivityResponse | null>(null);

  // What-If mutation
  const whatIfMutation = useMutation({
    mutationFn: () =>
      simulationService.runWhatIf({
        baselineGraph: getGraph(),
        modifications,
        algorithm: algorithm as Algorithm,
        options: {
          compareWithBaseline: true,
          calculateCostImpact: true,
          findNewBottlenecks: true,
          returnModifiedGraph: true,
        },
      }),
    // Явно указываем тип result
    onSuccess: (result: WhatIfResponse) => {
      setWhatIfResult(result);
      if (result.success) {
        toast.success("What-If анализ завершён");
      } else {
        toast.error(result.errorMessage || "Ошибка");
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  // Monte Carlo mutation
  const mcMutation = useMutation({
    mutationFn: () => {
      const uncertainties = edges.map((edge) => ({
        edge: { from: edge.from, to: edge.to },
        nodeId: edge.from,
        target: ModificationTarget.CAPACITY,
        distribution: {
          type: DistributionType.NORMAL,
          param1: edge.capacity,
          param2: edge.capacity * 0.2,
          param3: 0,
        },
      })) as unknown as UncertaintySpec[];

      return simulationService.runMonteCarlo({
        graph: getGraph(),
        config: {
          numIterations: mcIterations,
          confidenceLevel: mcConfidence,
          parallel: true,
          randomSeed: 0n,
        },
        uncertainties,
        algorithm: algorithm as Algorithm,
      });
    },
    // Явно указываем тип result
    onSuccess: (result: MonteCarloResponse) => {
      setMcResult(result);
      if (result.success) {
        toast.success("Monte Carlo симуляция завершена");
      } else {
        toast.error(result.errorMessage || "Ошибка");
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  // Sensitivity mutation
  const sensitivityMutation = useMutation({
    mutationFn: () => {
      const topEdges = [...edges]
        .sort((a, b) => b.capacity - a.capacity)
        .slice(0, 5);

      const parameters = topEdges.map((edge) => ({
        edge: { from: edge.from, to: edge.to },
        nodeId: 0n,
        target: ModificationTarget.CAPACITY,
        minMultiplier: 0.5,
        maxMultiplier: 1.5,
        numSteps: 10,
      })) as unknown as SensitivityParameter[];

      return simulationService.analyzeSensitivity({
        graph: getGraph(),
        parameters,
        algorithm: algorithm as Algorithm,
      });
    },
    // Явно указываем тип result
    onSuccess: (result: SensitivityResponse) => {
      setSensitivityResult(result);
      if (result.success) {
        toast.success("Анализ чувствительности завершён");
      } else {
        toast.error(result.errorMessage || "Ошибка");
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const addModification = () => {
    if (edges.length === 0) {
      toast.error("Нет рёбер для модификации");
      return;
    }
    const edge = edges[0];
    const newMod = {
      type: ModificationType.UPDATE_EDGE,
      edgeKey: { from: edge.from, to: edge.to },
      nodeId: 0n,
      target: ModificationTarget.CAPACITY,
      value: 1.2,
      isRelative: true,
      description: "Увеличение на 20%",
    } as unknown as Modification;
    setModifications([...modifications, newMod]);
  };

  const removeModification = (index: number) => {
    setModifications(modifications.filter((_, i) => i !== index));
  };

  const hasGraph = nodes.length > 0;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Симуляция</h1>

      {!hasGraph && (
        <Card className="bg-yellow-50 border-yellow-200">
          <p className="text-yellow-800">
            Сначала создайте граф в редакторе сети
          </p>
        </Card>
      )}

      <Tab.Group selectedIndex={activeTab} onChange={setActiveTab}>
        <Tab.List className="flex gap-2 border-b border-gray-200">
          {["What-If анализ", "Monte Carlo", "Чувствительность"].map((tab) => (
            <Tab
              key={tab}
              className={({ selected }) =>
                clsx(
                  "px-4 py-2 text-sm font-medium border-b-2 -mb-px outline-none",
                  selected
                    ? "border-primary-500 text-primary-600"
                    : "border-transparent text-gray-500 hover:text-gray-700",
                )
              }
            >
              {tab}
            </Tab>
          ))}
        </Tab.List>

        <Tab.Panels>
          {/* What-If Panel */}
          <Tab.Panel className="space-y-4">
            <Card>
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold">Модификации</h2>
                <Button variant="secondary" size="sm" onClick={addModification}>
                  + Добавить
                </Button>
              </div>

              {modifications.length === 0 ? (
                <p className="text-gray-500">
                  Добавьте модификации для анализа
                </p>
              ) : (
                <div className="space-y-2">
                  {modifications.map((mod, index) => (
                    <div
                      key={index}
                      className="flex items-center justify-between p-3 bg-gray-50 rounded"
                    >
                      <div>
                        <span className="font-medium">
                          Ребро {mod.edgeKey?.from?.toString()} →{" "}
                          {mod.edgeKey?.to?.toString()}
                        </span>
                        <span className="text-gray-500 ml-2">
                          {mod.isRelative ? `×${mod.value}` : mod.value}
                        </span>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeModification(index)}
                      >
                        Удалить
                      </Button>
                    </div>
                  ))}
                </div>
              )}

              <Button
                onClick={() => whatIfMutation.mutate()}
                loading={whatIfMutation.isPending}
                disabled={modifications.length === 0 || !hasGraph}
                className="mt-4"
              >
                Запустить What-If
              </Button>
            </Card>

            {whatIfResult?.success && whatIfResult.comparison && (
              <div className="grid grid-cols-2 gap-4">
                <Card>
                  <h3 className="font-medium mb-2">Базовый сценарий</h3>
                  <p className="text-2xl font-bold">
                    {whatIfResult.baseline?.maxFlow}
                  </p>
                  <p className="text-gray-500">Max Flow</p>
                </Card>
                <Card>
                  <h3 className="font-medium mb-2">После модификаций</h3>
                  <p className="text-2xl font-bold">
                    {whatIfResult.modified?.maxFlow}
                  </p>
                  <p
                    className={clsx(
                      whatIfResult.comparison.flowChangePercent > 0
                        ? "text-green-600"
                        : "text-red-600",
                    )}
                  >
                    {whatIfResult.comparison.flowChangePercent > 0 ? "+" : ""}
                    {whatIfResult.comparison.flowChangePercent.toFixed(1)}%
                  </p>
                </Card>
                <Card className="col-span-2">
                  <h3 className="font-medium mb-2">Сравнение</h3>
                  <p className="text-gray-600">
                    {whatIfResult.comparison.impactSummary}
                  </p>
                  <span
                    className={clsx(
                      "inline-block px-2 py-1 rounded text-sm mt-2",
                      IMPACT_COLORS[whatIfResult.comparison.impactLevel] ??
                        "bg-gray-100",
                    )}
                  >
                    {ImpactLevel[whatIfResult.comparison.impactLevel]}
                  </span>
                </Card>
              </div>
            )}
          </Tab.Panel>

          {/* Monte Carlo Panel */}
          <Tab.Panel className="space-y-4">
            <Card>
              <CardHeader title="Настройки Monte Carlo" />

              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="Количество итераций"
                  type="number"
                  value={mcIterations}
                  onChange={(e) => setMcIterations(Number(e.target.value))}
                  min={100}
                  max={100000}
                />
                <Select
                  label="Доверительный интервал"
                  value={mcConfidence}
                  onChange={(e) => setMcConfidence(Number(e.target.value))}
                  options={[
                    { value: 0.9, label: "90%" },
                    { value: 0.95, label: "95%" },
                    { value: 0.99, label: "99%" },
                  ]}
                />
              </div>

              <p className="text-sm text-gray-500 mt-2">
                Симуляция учитывает ±20% вариацию пропускной способности всех
                рёбер
              </p>

              <Button
                onClick={() => mcMutation.mutate()}
                loading={mcMutation.isPending}
                disabled={edges.length === 0}
                className="mt-4"
              >
                Запустить Monte Carlo
              </Button>
            </Card>

            {mcResult?.success && mcResult.flowStats && (
              <div className="grid grid-cols-2 gap-4">
                <Card className="col-span-2">
                  <CardHeader title="Распределение потока" />
                  <div className="grid grid-cols-4 gap-4 mb-4">
                    <div>
                      <p className="text-sm text-gray-500">Среднее</p>
                      <p className="text-xl font-bold">
                        {mcResult.flowStats.mean.toFixed(2)}
                      </p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Std Dev</p>
                      <p className="text-xl font-bold">
                        {mcResult.flowStats.stdDev.toFixed(2)}
                      </p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Min</p>
                      <p className="text-xl font-bold">
                        {mcResult.flowStats.min.toFixed(2)}
                      </p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Max</p>
                      <p className="text-xl font-bold">
                        {mcResult.flowStats.max.toFixed(2)}
                      </p>
                    </div>
                  </div>
                  <div className="p-4 bg-blue-50 rounded">
                    <p className="text-sm text-blue-800">
                      {(mcConfidence * 100).toFixed(0)}% доверительный интервал:{" "}
                      <strong>
                        [{mcResult.flowStats.confidenceIntervalLow.toFixed(2)},{" "}
                        {mcResult.flowStats.confidenceIntervalHigh.toFixed(2)}]
                      </strong>
                    </p>
                  </div>
                </Card>

                {mcResult.riskAnalysis && (
                  <Card className="col-span-2">
                    <CardHeader title="Анализ рисков" />
                    <div className="grid grid-cols-3 gap-4">
                      <div>
                        <p className="text-sm text-gray-500">Worst Case</p>
                        <p className="text-xl font-bold text-red-600">
                          {mcResult.riskAnalysis.worstCaseFlow.toFixed(2)}
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-gray-500">VaR (5%)</p>
                        <p className="text-xl font-bold text-orange-600">
                          {mcResult.riskAnalysis.valueAtRisk.toFixed(2)}
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-gray-500">Best Case</p>
                        <p className="text-xl font-bold text-green-600">
                          {mcResult.riskAnalysis.bestCaseFlow.toFixed(2)}
                        </p>
                      </div>
                    </div>
                  </Card>
                )}
              </div>
            )}
          </Tab.Panel>

          {/* Sensitivity Panel */}
          <Tab.Panel className="space-y-4">
            <Card>
              <CardHeader title="Анализ чувствительности" />
              <p className="text-gray-600 mb-4">
                Анализируется влияние изменения пропускной способности на
                максимальный поток для топ-5 рёбер по capacity (диапазон ±50%).
              </p>
              <Button
                onClick={() => sensitivityMutation.mutate()}
                loading={sensitivityMutation.isPending}
                disabled={edges.length === 0}
              >
                Запустить анализ
              </Button>
            </Card>

            {sensitivityResult?.success &&
              sensitivityResult.results.length > 0 && (
                <>
                  <Card>
                    <CardHeader title="Кривые чувствительности" />
                    <SensitivityLineChart
                      data={sensitivityResult.results[0].curve.map((p) => ({
                        parameter: `${(p.parameterValue * 100).toFixed(0)}%`,
                        flow: p.flowValue,
                        cost: p.costValue,
                      }))}
                      height={300}
                    />
                  </Card>

                  <Card>
                    <CardHeader title="Рейтинг влияния" />
                    <div className="space-y-2">
                      {sensitivityResult.rankings.map((r) => (
                        <div
                          key={r.parameterId}
                          className="flex items-center justify-between p-3 bg-gray-50 rounded"
                        >
                          <div>
                            <span className="font-medium">#{r.rank}</span>
                            <span className="text-gray-500 ml-2">
                              {r.description}
                            </span>
                          </div>
                          <div className="text-right">
                            <p className="font-bold">
                              Индекс: {r.sensitivityIndex.toFixed(3)}
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </Card>
                </>
              )}
          </Tab.Panel>
        </Tab.Panels>
      </Tab.Group>
    </div>
  );
}
