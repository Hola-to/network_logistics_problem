import { useState, useCallback } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Tab } from "@headlessui/react";
import { create } from "@bufbuild/protobuf";
import { PlusIcon, TrashIcon } from "@heroicons/react/24/outline";
import clsx from "clsx";
import Card, { CardHeader } from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";
import Select from "@/components/ui/Select";
import Modal from "@/components/ui/Modal";
import { useGraphStore } from "@/stores/graphStore";
import { simulationService } from "@/api/services";
import { SensitivityLineChart } from "@/components/visual/FlowChart";
import { Algorithm } from "@gen/logistics/common/v1/common_pb";
import {
  ModificationType,
  ModificationTarget,
  DistributionType,
  ImpactLevel,
  ModificationSchema,
  UncertaintySpecSchema,
  DistributionSchema,
  SensitivityParameterSchema,
} from "@gen/logistics/gateway/v1/gateway_pb";
import type {
  Modification,
  WhatIfResponse,
  MonteCarloResponse,
  SensitivityResponse,
} from "@gen/logistics/gateway/v1/gateway_pb";
import { EdgeKeySchema } from "@gen/logistics/common/v1/common_pb";

// ============================================================================
// Helpers для создания protobuf messages
// ============================================================================

const createEdgeKey = (from: bigint, to: bigint) => {
  return create(EdgeKeySchema, { from, to }) as unknown as {
    from: bigint;
    to: bigint;
  };
};

const createModification = (data: {
  type: ModificationType;
  from: bigint;
  to: bigint;
  target: ModificationTarget;
  value: number;
  isRelative: boolean;
  description: string;
}): Modification => {
  return create(ModificationSchema, {
    type: data.type,
    edgeKey: createEdgeKey(data.from, data.to),
    nodeId: 0n,
    target: data.target,
    value: data.value,
    isRelative: data.isRelative,
    description: data.description,
  }) as unknown as Modification;
};

const createUncertaintySpec = (data: {
  from: bigint;
  to: bigint;
  target: ModificationTarget;
  mean: number;
  stdDev: number;
}) => {
  return create(UncertaintySpecSchema, {
    edge: createEdgeKey(data.from, data.to),
    nodeId: 0n,
    target: data.target,
    distribution: create(DistributionSchema, {
      type: DistributionType.NORMAL,
      param1: data.mean,
      param2: data.stdDev,
      param3: 0,
    }),
  });
};

const createSensitivityParameter = (data: {
  from: bigint;
  to: bigint;
  target: ModificationTarget;
  minMultiplier: number;
  maxMultiplier: number;
  numSteps: number;
}) => {
  return create(SensitivityParameterSchema, {
    edge: createEdgeKey(data.from, data.to),
    nodeId: 0n,
    target: data.target,
    minMultiplier: data.minMultiplier,
    maxMultiplier: data.maxMultiplier,
    numSteps: data.numSteps,
  });
};

// ============================================================================
// Константы
// ============================================================================

const IMPACT_COLORS: Record<number, string> = {
  [ImpactLevel.NONE]: "bg-gray-100 text-gray-800",
  [ImpactLevel.LOW]: "bg-green-100 text-green-800",
  [ImpactLevel.MEDIUM]: "bg-yellow-100 text-yellow-800",
  [ImpactLevel.HIGH]: "bg-orange-100 text-orange-800",
  [ImpactLevel.CRITICAL]: "bg-red-100 text-red-800",
};

const TARGET_OPTIONS = [
  { value: ModificationTarget.CAPACITY, label: "Пропускная способность" },
  { value: ModificationTarget.COST, label: "Стоимость" },
  { value: ModificationTarget.LENGTH, label: "Длина" },
];

const TARGET_NAMES: Record<number, string> = {
  [ModificationTarget.CAPACITY]: "Capacity",
  [ModificationTarget.COST]: "Cost",
  [ModificationTarget.LENGTH]: "Length",
};

// ============================================================================
// Интерфейс для локальных модификаций
// ============================================================================

interface LocalModification {
  id: string;
  from: bigint;
  to: bigint;
  target: ModificationTarget;
  value: number;
  isRelative: boolean;
  description: string;
}

// ============================================================================
// Модальное окно добавления модификации
// ============================================================================

interface AddModificationModalProps {
  open: boolean;
  onClose: () => void;
  edges: Array<{ from: bigint; to: bigint; capacity: number; cost?: number }>;
  nodes: Array<{ id: bigint; name?: string }>;
  onAdd: (mod: LocalModification) => void;
}

function AddModificationModal({
  open,
  onClose,
  edges,
  nodes,
  onAdd,
}: AddModificationModalProps) {
  const [selectedEdge, setSelectedEdge] = useState<string>("");
  const [target, setTarget] = useState<ModificationTarget>(
    ModificationTarget.CAPACITY,
  );
  const [isRelative, setIsRelative] = useState(true);
  const [value, setValue] = useState(1.2);
  const [description, setDescription] = useState("");

  // Получаем имена узлов для отображения
  const getNodeName = (id: bigint) => {
    const node = nodes.find((n) => n.id === id);
    return node?.name || `Узел ${id}`;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!selectedEdge) {
      toast.error("Выберите ребро");
      return;
    }

    const [fromStr, toStr] = selectedEdge.split("-");
    const from = BigInt(fromStr);
    const to = BigInt(toStr);

    const mod: LocalModification = {
      id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      from,
      to,
      target,
      value,
      isRelative,
      description: description || (isRelative ? `×${value}` : `=${value}`),
    };

    onAdd(mod);

    // Reset form
    setSelectedEdge("");
    setValue(1.2);
    setDescription("");
    onClose();
  };

  const handleClose = () => {
    setSelectedEdge("");
    setValue(1.2);
    setDescription("");
    onClose();
  };

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Добавить модификацию"
      size="md"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Выбор ребра */}
        <div>
          <label className="label">Ребро</label>
          {edges.length === 0 ? (
            <p className="text-sm text-red-500">
              Нет доступных рёбер. Создайте граф в редакторе.
            </p>
          ) : (
            <select
              value={selectedEdge}
              onChange={(e) => setSelectedEdge(e.target.value)}
              className="input"
              required
            >
              <option value="">Выберите ребро...</option>
              {edges.map((e) => (
                <option key={`${e.from}-${e.to}`} value={`${e.from}-${e.to}`}>
                  {getNodeName(e.from)} → {getNodeName(e.to)} (cap: {e.capacity}
                  , cost: {e.cost ?? 0})
                </option>
              ))}
            </select>
          )}
        </div>

        {/* Выбор параметра */}
        <Select
          label="Параметр для изменения"
          value={target}
          onChange={(e) => setTarget(Number(e.target.value))}
          options={TARGET_OPTIONS.map((o) => ({
            value: o.value,
            label: o.label,
          }))}
        />

        {/* Тип изменения */}
        <div>
          <label className="label">Тип изменения</label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                checked={isRelative}
                onChange={() => {
                  setIsRelative(true);
                  setValue(1.2);
                }}
                className="text-primary-600"
              />
              <span>Множитель (×)</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                checked={!isRelative}
                onChange={() => {
                  setIsRelative(false);
                  setValue(10);
                }}
                className="text-primary-600"
              />
              <span>Абсолютное значение</span>
            </label>
          </div>
        </div>

        {/* Значение */}
        <Input
          label={isRelative ? "Множитель" : "Новое значение"}
          type="number"
          value={value}
          onChange={(e) => setValue(Number(e.target.value))}
          min={isRelative ? 0.01 : 0}
          step={isRelative ? 0.1 : 1}
          hint={
            isRelative
              ? "1.0 = без изменений, 1.5 = +50%, 0.5 = -50%"
              : "Новое абсолютное значение параметра"
          }
        />

        {/* Описание */}
        <Input
          label="Описание (опционально)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Например: Расширение дороги A→B"
        />

        {/* Кнопки */}
        <div className="flex gap-2 pt-2 border-t">
          <Button type="submit" className="flex-1" disabled={!selectedEdge}>
            Добавить
          </Button>
          <Button type="button" variant="secondary" onClick={handleClose}>
            Отмена
          </Button>
        </div>
      </form>
    </Modal>
  );
}

// ============================================================================
// Главный компонент
// ============================================================================

export default function Simulation() {
  const { getGraph, algorithm, nodes, edges, flowResult } = useGraphStore();
  const [activeTab, setActiveTab] = useState(0);

  // What-If state
  const [modifications, setModifications] = useState<LocalModification[]>([]);
  const [whatIfResult, setWhatIfResult] = useState<WhatIfResponse | null>(null);
  const [showAddMod, setShowAddMod] = useState(false);

  // Monte Carlo state
  const [mcIterations, setMcIterations] = useState(1000);
  const [mcConfidence, setMcConfidence] = useState(0.95);
  const [mcResult, setMcResult] = useState<MonteCarloResponse | null>(null);

  // Sensitivity state
  const [sensitivityResult, setSensitivityResult] =
    useState<SensitivityResponse | null>(null);
  const [sensMinMult, setSensMinMult] = useState(0.5);
  const [sensMaxMult, setSensMaxMult] = useState(1.5);
  const [sensSteps, setSensSteps] = useState(10);
  const [sensTopN, setSensTopN] = useState(5);

  const hasGraph = nodes.length > 0 && edges.length > 0;
  const hasSolution = flowResult !== null;

  // Получить имя узла
  const getNodeName = useCallback(
    (id: bigint) => {
      const node = nodes.find((n) => n.id === id);
      return node?.name || `Узел ${id}`;
    },
    [nodes],
  );

  // Добавить модификацию
  const handleAddModification = useCallback((mod: LocalModification) => {
    setModifications((prev) => [...prev, mod]);
    toast.success("Модификация добавлена");
  }, []);

  // Удалить модификацию
  const removeModification = (id: string) => {
    setModifications((prev) => prev.filter((m) => m.id !== id));
  };

  // Очистить все модификации
  const clearModifications = () => {
    setModifications([]);
    setWhatIfResult(null);
  };

  // What-If mutation
  const whatIfMutation = useMutation({
    mutationFn: async () => {
      if (modifications.length === 0) {
        throw new Error("Добавьте хотя бы одну модификацию");
      }

      const graph = getGraph();

      // Конвертируем локальные модификации в protobuf
      const protoModifications = modifications.map((m) =>
        createModification({
          type: ModificationType.UPDATE_EDGE,
          from: m.from,
          to: m.to,
          target: m.target,
          value: m.value,
          isRelative: m.isRelative,
          description: m.description,
        }),
      );

      return simulationService.runWhatIf({
        baselineGraph: graph,
        modifications: protoModifications,
        algorithm: algorithm as Algorithm,
        options: {
          compareWithBaseline: true,
          calculateCostImpact: true,
          findNewBottlenecks: true,
          returnModifiedGraph: true,
        },
      });
    },
    onSuccess: (result: WhatIfResponse) => {
      setWhatIfResult(result);
      if (result.success) {
        toast.success("What-If анализ завершён");
      } else {
        toast.error(result.errorMessage || "Ошибка анализа");
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  // Monte Carlo mutation
  const mcMutation = useMutation({
    mutationFn: async () => {
      const graph = getGraph();

      const uncertainties = edges.map((edge) =>
        createUncertaintySpec({
          from: edge.from,
          to: edge.to,
          target: ModificationTarget.CAPACITY,
          mean: edge.capacity,
          stdDev: edge.capacity * 0.2,
        }),
      );

      return simulationService.runMonteCarlo({
        graph,
        config: {
          numIterations: mcIterations,
          confidenceLevel: mcConfidence,
          parallel: true,
          randomSeed: 0n,
        },
        uncertainties: uncertainties as any,
        algorithm: algorithm as Algorithm,
      });
    },
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
    mutationFn: async () => {
      const graph = getGraph();

      const topEdges = [...edges]
        .sort((a, b) => b.capacity - a.capacity)
        .slice(0, sensTopN);

      const parameters = topEdges.map((edge) =>
        createSensitivityParameter({
          from: edge.from,
          to: edge.to,
          target: ModificationTarget.CAPACITY,
          minMultiplier: sensMinMult,
          maxMultiplier: sensMaxMult,
          numSteps: sensSteps,
        }),
      );

      return simulationService.analyzeSensitivity({
        graph,
        parameters: parameters as any,
        algorithm: algorithm as Algorithm,
      });
    },
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

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Симуляция</h1>

      {!hasGraph && (
        <Card className="bg-yellow-50 border-yellow-200">
          <p className="text-yellow-800">
            ⚠️ Сначала создайте граф в{" "}
            <a href="/network" className="underline font-medium">
              редакторе сети
            </a>
          </p>
        </Card>
      )}

      {hasGraph && !hasSolution && (
        <Card className="bg-blue-50 border-blue-200">
          <p className="text-blue-800">
            💡 Рекомендуется сначала выполнить оптимизацию в редакторе для
            получения baseline
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
                  "px-4 py-2 text-sm font-medium border-b-2 -mb-px outline-none transition-colors",
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
          {/* ================ What-If Panel ================ */}
          <Tab.Panel className="space-y-4">
            <Card>
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h2 className="text-lg font-semibold">Модификации</h2>
                  <p className="text-sm text-gray-500">
                    Добавьте изменения параметров рёбер для анализа
                  </p>
                </div>
                <div className="flex gap-2">
                  {modifications.length > 0 && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={clearModifications}
                    >
                      Очистить все
                    </Button>
                  )}
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setShowAddMod(true)}
                    disabled={edges.length === 0}
                    icon={<PlusIcon className="w-4 h-4" />}
                  >
                    Добавить
                  </Button>
                </div>
              </div>

              {modifications.length === 0 ? (
                <div className="text-center py-8 bg-gray-50 rounded-lg">
                  <p className="text-gray-500 mb-2">Нет модификаций</p>
                  <p className="text-sm text-gray-400">
                    Нажмите "Добавить" чтобы создать модификацию для анализа
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {modifications.map((mod) => (
                    <div
                      key={mod.id}
                      className="flex items-center justify-between p-3 bg-gray-50 rounded-lg border border-gray-200"
                    >
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">
                            {getNodeName(mod.from)} → {getNodeName(mod.to)}
                          </span>
                          <span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">
                            {TARGET_NAMES[mod.target] || "Unknown"}
                          </span>
                        </div>
                        <div className="text-sm text-gray-500 mt-1">
                          {mod.isRelative ? (
                            <span>
                              Умножить на{" "}
                              <strong className="text-primary-600">
                                {mod.value}
                              </strong>
                              {mod.value > 1
                                ? ` (+${((mod.value - 1) * 100).toFixed(0)}%)`
                                : ` (${((mod.value - 1) * 100).toFixed(0)}%)`}
                            </span>
                          ) : (
                            <span>
                              Установить в{" "}
                              <strong className="text-primary-600">
                                {mod.value}
                              </strong>
                            </span>
                          )}
                          {mod.description &&
                            mod.description !== `×${mod.value}` &&
                            mod.description !== `=${mod.value}` && (
                              <span className="text-gray-400 ml-2">
                                — {mod.description}
                              </span>
                            )}
                        </div>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeModification(mod.id)}
                        className="text-red-500 hover:bg-red-50"
                      >
                        <TrashIcon className="w-4 h-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}

              <div className="mt-4 pt-4 border-t">
                <Button
                  onClick={() => whatIfMutation.mutate()}
                  loading={whatIfMutation.isPending}
                  disabled={modifications.length === 0 || !hasGraph}
                  className="w-full"
                >
                  Запустить What-If анализ ({modifications.length} модификаций)
                </Button>
              </div>
            </Card>

            {/* Результаты What-If */}
            {whatIfResult?.success && whatIfResult.comparison && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Card>
                  <h3 className="font-medium mb-2 text-gray-600">
                    Базовый сценарий
                  </h3>
                  <p className="text-3xl font-bold text-gray-700">
                    {whatIfResult.baseline?.maxFlow?.toFixed(1)}
                  </p>
                  <p className="text-gray-500 text-sm">Max Flow</p>
                </Card>
                <Card>
                  <h3 className="font-medium mb-2 text-gray-600">
                    После модификаций
                  </h3>
                  <p className="text-3xl font-bold text-primary-600">
                    {whatIfResult.modified?.maxFlow?.toFixed(1)}
                  </p>
                  <p
                    className={clsx(
                      "text-sm font-medium",
                      whatIfResult.comparison.flowChangePercent > 0
                        ? "text-green-600"
                        : whatIfResult.comparison.flowChangePercent < 0
                          ? "text-red-600"
                          : "text-gray-500",
                    )}
                  >
                    {whatIfResult.comparison.flowChangePercent > 0 ? "+" : ""}
                    {whatIfResult.comparison.flowChangePercent.toFixed(1)}%
                  </p>
                </Card>
                <Card className="md:col-span-2">
                  <h3 className="font-medium mb-2">Сравнение</h3>
                  <p className="text-gray-600">
                    {whatIfResult.comparison.impactSummary}
                  </p>
                  <div className="flex items-center gap-2 mt-2">
                    <span className="text-sm text-gray-500">
                      Уровень влияния:
                    </span>
                    <span
                      className={clsx(
                        "px-2 py-1 rounded text-sm font-medium",
                        IMPACT_COLORS[whatIfResult.comparison.impactLevel] ??
                          "bg-gray-100",
                      )}
                    >
                      {ImpactLevel[whatIfResult.comparison.impactLevel]}
                    </span>
                  </div>
                </Card>
              </div>
            )}
          </Tab.Panel>

          {/* ================ Monte Carlo Panel ================ */}
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
                Симуляция учитывает ±20% вариацию capacity всех рёбер
                (нормальное распределение)
              </p>
              <Button
                onClick={() => mcMutation.mutate()}
                loading={mcMutation.isPending}
                disabled={!hasGraph}
                className="mt-4"
              >
                Запустить Monte Carlo
              </Button>
            </Card>

            {mcResult?.success && mcResult.flowStats && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Card className="md:col-span-2">
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
                  <Card className="md:col-span-2">
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

          {/* ================ Sensitivity Panel ================ */}
          <Tab.Panel className="space-y-4">
            <Card>
              <CardHeader title="Анализ чувствительности" />
              <p className="text-gray-600 mb-4">
                Анализируется влияние изменения capacity на максимальный поток
              </p>

              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
                <Input
                  label="Мин. множитель"
                  type="number"
                  value={sensMinMult}
                  onChange={(e) => setSensMinMult(Number(e.target.value))}
                  min={0.1}
                  max={1}
                  step={0.1}
                />
                <Input
                  label="Макс. множитель"
                  type="number"
                  value={sensMaxMult}
                  onChange={(e) => setSensMaxMult(Number(e.target.value))}
                  min={1}
                  max={3}
                  step={0.1}
                />
                <Input
                  label="Шагов"
                  type="number"
                  value={sensSteps}
                  onChange={(e) => setSensSteps(Number(e.target.value))}
                  min={5}
                  max={50}
                />
                <Input
                  label="Топ N рёбер"
                  type="number"
                  value={sensTopN}
                  onChange={(e) => setSensTopN(Number(e.target.value))}
                  min={1}
                  max={Math.min(10, edges.length || 1)}
                />
              </div>

              <Button
                onClick={() => sensitivityMutation.mutate()}
                loading={sensitivityMutation.isPending}
                disabled={!hasGraph}
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
                    <CardHeader title="Рейтинг влияния параметров" />
                    <div className="space-y-2">
                      {sensitivityResult.rankings.map((r) => (
                        <div
                          key={r.parameterId}
                          className="flex items-center justify-between p-3 bg-gray-50 rounded"
                        >
                          <div>
                            <span className="font-medium">#{r.rank}</span>
                            <span className="text-gray-500 ml-2">
                              {r.description || r.parameterId}
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

      {/* Модальное окно добавления модификации */}
      <AddModificationModal
        open={showAddMod}
        onClose={() => setShowAddMod(false)}
        edges={edges}
        nodes={nodes}
        onAdd={handleAddModification}
      />
    </div>
  );
}
