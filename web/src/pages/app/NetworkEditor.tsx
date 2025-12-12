import { useState, useCallback, useMemo } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import {
  PlayIcon,
  TrashIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  Cog6ToothIcon,
  PlusIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  BookmarkIcon,
  InformationCircleIcon,
  CurrencyDollarIcon,
} from "@heroicons/react/24/outline";
import { Link } from "react-router-dom";
import GraphCanvas from "@/components/visual/GraphCanvas";
import Card from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";
import Modal from "@/components/ui/Modal";
import Badge from "@/components/ui/Badge";
import { useGraphStore } from "@/stores/graphStore";
import { solverService, historyService } from "@/api/services";
import { NodeType, Algorithm } from "@gen/logistics/common/v1/common_pb";
import clsx from "clsx";
import type {
  SolveGraphResponse,
  SaveCalculationResponse,
} from "@gen/logistics/gateway/v1/gateway_pb";

// ============================================================================
// Конфигурация
// ============================================================================

const NODE_TYPES_CONFIG = [
  {
    type: NodeType.SOURCE,
    label: "Источник",
    icon: "🟢",
    color: "bg-green-500",
    description: "Начальная точка потока",
    unique: true,
  },
  {
    type: NodeType.SINK,
    label: "Сток",
    icon: "🔴",
    color: "bg-red-500",
    description: "Конечная точка потока",
    unique: true,
  },
  {
    type: NodeType.WAREHOUSE,
    label: "Склад",
    icon: "📦",
    color: "bg-blue-500",
    description: "Промежуточное хранилище",
    unique: false,
  },
  {
    type: NodeType.DELIVERY_POINT,
    label: "Точка доставки",
    icon: "📍",
    color: "bg-orange-500",
    description: "Пункт назначения",
    unique: false,
  },
  {
    type: NodeType.INTERSECTION,
    label: "Узел",
    icon: "⚫",
    color: "bg-gray-500",
    description: "Транзитная точка",
    unique: false,
  },
];

const ALGORITHMS = [
  {
    value: Algorithm.DINIC,
    label: "Dinic",
    description: "Рекомендуется для большинства задач",
    supportsCost: false,
  },
  {
    value: Algorithm.EDMONDS_KARP,
    label: "Edmonds-Karp",
    description: "Классический BFS-алгоритм",
    supportsCost: false,
  },
  {
    value: Algorithm.PUSH_RELABEL,
    label: "Push-Relabel",
    description: "Для очень плотных графов",
    supportsCost: false,
  },
  {
    value: Algorithm.MIN_COST,
    label: "Min-Cost Flow",
    description: "Минимизация стоимости доставки",
    supportsCost: true,
  },
  {
    value: Algorithm.FORD_FULKERSON,
    label: "Ford-Fulkerson",
    description: "Классический алгоритм (обучение)",
    supportsCost: false,
  },
];

// ============================================================================
// Хук для определения поддержки стоимости
// ============================================================================

function useAlgorithmSupport(algorithm: Algorithm) {
  return useMemo(() => {
    const algoConfig = ALGORITHMS.find((a) => a.value === algorithm);
    return {
      supportsCost: algoConfig?.supportsCost ?? false,
      algorithmName: algoConfig?.label ?? "Unknown",
    };
  }, [algorithm]);
}

// ============================================================================
// Компонент палитры узлов
// ============================================================================

interface NodePaletteProps {
  onAddNode: (type: NodeType) => void;
  disabled?: boolean;
  hasSource: boolean;
  hasSink: boolean;
}

function NodePalette({
  onAddNode,
  disabled,
  hasSource,
  hasSink,
}: NodePaletteProps) {
  const handleAdd = (config: (typeof NODE_TYPES_CONFIG)[0]) => {
    if (config.type === NodeType.SOURCE && hasSource) {
      toast.error("Источник уже существует. Можно добавить только один.");
      return;
    }
    if (config.type === NodeType.SINK && hasSink) {
      toast.error("Сток уже существует. Можно добавить только один.");
      return;
    }
    onAddNode(config.type);
  };

  return (
    <Card>
      <h3 className="font-medium mb-3 flex items-center gap-2">
        <PlusIcon className="w-4 h-4" />
        Добавить узел
      </h3>
      <div className="grid grid-cols-1 gap-2">
        {NODE_TYPES_CONFIG.map((config) => {
          const isDisabled =
            disabled ||
            (config.type === NodeType.SOURCE && hasSource) ||
            (config.type === NodeType.SINK && hasSink);

          const isAdded =
            (config.type === NodeType.SOURCE && hasSource) ||
            (config.type === NodeType.SINK && hasSink);

          return (
            <button
              key={config.type}
              onClick={() => handleAdd(config)}
              disabled={isDisabled}
              className={clsx(
                "flex items-center gap-3 p-3 rounded-lg border-2 transition-all text-left",
                isAdded
                  ? "border-green-300 bg-green-50 cursor-default"
                  : isDisabled
                    ? "border-gray-200 bg-gray-50 opacity-50 cursor-not-allowed"
                    : "border-dashed border-gray-200 bg-white hover:border-primary-400 hover:bg-primary-50",
              )}
            >
              <div
                className={clsx(
                  "w-8 h-8 rounded-full flex items-center justify-center text-white text-sm",
                  config.color,
                )}
              >
                {config.icon}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="font-medium text-gray-900">{config.label}</p>
                  {isAdded && (
                    <CheckCircleIcon className="w-4 h-4 text-green-500" />
                  )}
                </div>
                <p className="text-xs text-gray-500 truncate">
                  {config.description}
                </p>
              </div>
            </button>
          );
        })}
      </div>
    </Card>
  );
}

// ============================================================================
// Модальное окно добавления ребра
// ============================================================================

interface AddEdgeModalProps {
  open: boolean;
  onClose: () => void;
  nodes: Array<{ id: bigint; name?: string }>;
  onAdd: (from: bigint, to: bigint, capacity: number, cost: number) => void;
  supportsCost: boolean;
}

function AddEdgeModal({
  open,
  onClose,
  nodes,
  onAdd,
  supportsCost,
}: AddEdgeModalProps) {
  const [fromId, setFromId] = useState<string>("");
  const [toId, setToId] = useState<string>("");
  const [capacity, setCapacity] = useState(10);
  const [cost, setCost] = useState(1);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!fromId || !toId) {
      toast.error("Выберите оба узла");
      return;
    }
    if (fromId === toId) {
      toast.error("Узлы должны быть разными");
      return;
    }
    // Всегда передаём cost - данные сохраняются независимо от алгоритма
    onAdd(BigInt(fromId), BigInt(toId), capacity, cost);
    onClose();
    setFromId("");
    setToId("");
    setCapacity(10);
    setCost(1);
  };

  return (
    <Modal open={open} onClose={onClose} title="Добавить ребро" size="sm">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="label">Из узла</label>
            <select
              value={fromId}
              onChange={(e) => setFromId(e.target.value)}
              className="input"
              required
            >
              <option value="">Выберите...</option>
              {nodes.map((n) => (
                <option key={String(n.id)} value={String(n.id)}>
                  {n.name || `Узел ${n.id}`}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">В узел</label>
            <select
              value={toId}
              onChange={(e) => setToId(e.target.value)}
              className="input"
              required
            >
              <option value="">Выберите...</option>
              {nodes.map((n) => (
                <option key={String(n.id)} value={String(n.id)}>
                  {n.name || `Узел ${n.id}`}
                </option>
              ))}
            </select>
          </div>
        </div>

        <Input
          label="Пропускная способность"
          type="number"
          value={capacity}
          onChange={(e) => setCapacity(Number(e.target.value))}
          min={1}
          required
        />

        {/* Поле стоимости - показываем только для Min-Cost Flow */}
        {supportsCost ? (
          <Input
            label="Стоимость за единицу"
            type="number"
            value={cost}
            onChange={(e) => setCost(Number(e.target.value))}
            min={0}
            step={0.1}
            hint="Стоимость транспортировки одной единицы потока"
          />
        ) : (
          <div className="p-3 bg-gray-50 border border-gray-200 rounded-lg">
            <div className="flex items-start gap-2">
              <InformationCircleIcon className="w-5 h-5 text-gray-400 shrink-0 mt-0.5" />
              <div className="text-sm text-gray-600">
                <p className="font-medium">Стоимость не учитывается</p>
                <p className="text-gray-500 mt-1">
                  Выбранный алгоритм оптимизирует только поток. Для минимизации
                  затрат используйте{" "}
                  <span className="font-medium text-emerald-600">
                    Min-Cost Flow
                  </span>
                  .
                </p>
              </div>
            </div>
          </div>
        )}

        <div className="flex gap-2 pt-2">
          <Button type="submit" className="flex-1">
            Добавить
          </Button>
          <Button type="button" variant="secondary" onClick={onClose}>
            Отмена
          </Button>
        </div>
      </form>
    </Modal>
  );
}

// ============================================================================
// Компонент настроек алгоритма
// ============================================================================

interface AlgorithmSettingsProps {
  algorithm: Algorithm;
  onAlgorithmChange: (algorithm: Algorithm) => void;
}

function AlgorithmSettings({
  algorithm,
  onAlgorithmChange,
}: AlgorithmSettingsProps) {
  const { supportsCost } = useAlgorithmSupport(algorithm);

  return (
    <Card>
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-medium">Настройки алгоритма</h3>
        <Link
          to="/algorithms"
          className="text-xs text-primary-600 hover:text-primary-700"
        >
          Подробнее →
        </Link>
      </div>

      <div className="space-y-3">
        {ALGORITHMS.map((algo) => (
          <label
            key={algo.value}
            className={clsx(
              "flex items-start gap-3 p-3 rounded-lg border-2 cursor-pointer transition-all",
              algorithm === algo.value
                ? "border-primary-500 bg-primary-50"
                : "border-gray-200 hover:border-gray-300",
            )}
          >
            <input
              type="radio"
              name="algorithm"
              value={algo.value}
              checked={algorithm === algo.value}
              onChange={() => onAlgorithmChange(algo.value)}
              className="mt-1"
            />
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <span className="font-medium text-gray-900">{algo.label}</span>
                {algo.supportsCost && (
                  <Badge variant="success" size="sm">
                    <CurrencyDollarIcon className="w-3 h-3 mr-1" />
                    Cost
                  </Badge>
                )}
                {algo.value === Algorithm.DINIC && (
                  <Badge variant="info" size="sm">
                    Рекомендуется
                  </Badge>
                )}
              </div>
              <p className="text-xs text-gray-500 mt-1">{algo.description}</p>
            </div>
          </label>
        ))}
      </div>

      {/* Подсказка о стоимости */}
      <div
        className={clsx(
          "mt-4 p-3 rounded-lg border",
          supportsCost
            ? "bg-emerald-50 border-emerald-200"
            : "bg-gray-50 border-gray-200",
        )}
      >
        <div className="flex items-start gap-2">
          {supportsCost ? (
            <CurrencyDollarIcon className="w-5 h-5 text-emerald-600 shrink-0" />
          ) : (
            <InformationCircleIcon className="w-5 h-5 text-gray-400 shrink-0" />
          )}
          <div className="text-sm">
            {supportsCost ? (
              <>
                <p className="font-medium text-emerald-800">
                  Учёт стоимости включён
                </p>
                <p className="text-emerald-600 mt-1">
                  Алгоритм найдёт максимальный поток с минимальной общей
                  стоимостью.
                </p>
              </>
            ) : (
              <>
                <p className="font-medium text-gray-700">
                  Только максимальный поток
                </p>
                <p className="text-gray-500 mt-1">
                  Стоимость рёбер сохраняется, но не влияет на результат.
                  Переключитесь на Min-Cost Flow для оптимизации затрат.
                </p>
              </>
            )}
          </div>
        </div>
      </div>
    </Card>
  );
}

// ============================================================================
// Главный компонент
// ============================================================================

export default function NetworkEditor() {
  const queryClient = useQueryClient();

  const {
    nodes,
    edges,
    sourceId,
    sinkId,
    name,
    algorithm,
    flowResult,
    metrics,
    solvedGraph,
    selectedNodeId,
    selectedEdgeKey,
    isLoading,
    addNode,
    updateNode,
    removeNode,
    addEdge,
    updateEdge,
    removeEdge,
    setSourceSink,
    setName,
    setAlgorithm,
    setSolution,
    setLoading,
    getGraph,
    clearGraph,
    clearSolution,
    loadGraph,
    hasSolution,
  } = useGraphStore();

  const [showSettings, setShowSettings] = useState(false);
  const [showAddEdge, setShowAddEdge] = useState(false);
  const [autoSave, setAutoSave] = useState(true);
  const [isSaved, setIsSaved] = useState(false);

  const hasSource = sourceId !== null;
  const hasSink = sinkId !== null;
  const canSave = hasSolution();

  // Определяем поддержку стоимости для текущего алгоритма
  const { supportsCost, algorithmName } = useAlgorithmSupport(algorithm);

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: async () => {
      if (!flowResult) {
        throw new Error("Сначала выполните оптимизацию");
      }

      const graph = getGraph();

      return historyService.saveCalculation({
        name: name || "Безымянный расчёт",
        graph,
        flowResult,
        solvedGraph: solvedGraph ?? undefined,
        metrics,
      });
    },
    onSuccess: (_response: SaveCalculationResponse) => {
      toast.success(`Сохранено в историю`);
      setIsSaved(true);
      queryClient.invalidateQueries({ queryKey: ["calculations"] });
      queryClient.invalidateQueries({ queryKey: ["statistics"] });
    },
    onError: (error: Error) => {
      console.error("Save error:", error);
      toast.error(`Ошибка сохранения: ${error.message}`);
    },
  });

  // Solve mutation
  const solveMutation = useMutation({
    mutationFn: () => {
      if (sourceId === null || sinkId === null) {
        return Promise.reject(new Error("Укажите источник и сток"));
      }
      return solverService.solve({
        graph: getGraph(),
        algorithm,
        options: { returnPaths: true },
      });
    },
    onMutate: () => {
      setLoading(true);
      setIsSaved(false);
    },
    onSuccess: async (response: SolveGraphResponse) => {
      if (response.success && response.result && response.solvedGraph) {
        setSolution(
          response.solvedGraph,
          response.result,
          response.metrics ?? null,
        );

        // Сообщение зависит от алгоритма
        if (supportsCost && response.result.totalCost > 0) {
          toast.success(
            `Макс. поток: ${response.result.maxFlow}, Мин. стоимость: ₽${response.result.totalCost.toFixed(2)}`,
          );
        } else {
          toast.success(
            `Найден максимальный поток: ${response.result.maxFlow}`,
          );
        }

        // Автосохранение
        if (autoSave) {
          try {
            const graph = getGraph();
            await historyService.saveCalculation({
              name: name || "Безымянный расчёт",
              graph,
              flowResult: response.result,
              solvedGraph: response.solvedGraph,
              metrics: response.metrics ?? null,
            });
            setIsSaved(true);
            queryClient.invalidateQueries({ queryKey: ["calculations"] });
            queryClient.invalidateQueries({ queryKey: ["statistics"] });
            toast.success("Автосохранено в историю");
          } catch (e) {
            console.error("Auto-save failed:", e);
          }
        }
      } else {
        toast.error(response.errorMessage || "Ошибка решения");
      }
    },
    onError: (error: Error) => toast.error(error.message),
    onSettled: () => setLoading(false),
  });

  // Ручное сохранение
  const handleManualSave = () => {
    if (!canSave) {
      toast.error("Сначала выполните оптимизацию");
      return;
    }
    saveMutation.mutate();
  };

  // Добавление узла
  const handleAddNodeOfType = useCallback(
    (type: NodeType) => {
      const offsetX = (nodes.length % 5) * 1.5;
      const offsetY = Math.floor(nodes.length / 5) * 1.5;
      const config = NODE_TYPES_CONFIG.find((c) => c.type === type);

      const newNode = addNode({
        x: 2 + offsetX,
        y: 2 + offsetY,
        type,
        name: `${config?.label} ${nodes.length + 1}`,
      });

      if (type === NodeType.SOURCE) {
        setSourceSink(newNode.id, sinkId);
        toast.success("Источник добавлен");
      } else if (type === NodeType.SINK) {
        setSourceSink(sourceId, newNode.id);
        toast.success("Сток добавлен");
      }

      clearSolution();
      setIsSaved(false);
    },
    [addNode, nodes.length, sourceId, sinkId, setSourceSink, clearSolution],
  );

  // Добавление ребра - всегда сохраняем cost
  const handleAddEdge = useCallback(
    (from: bigint, to: bigint, capacity: number, cost: number) => {
      const edge = addEdge({ from, to, capacity, cost });
      if (edge) {
        toast.success("Ребро добавлено");
        clearSolution();
        setIsSaved(false);
      } else {
        toast.error("Ребро уже существует");
      }
    },
    [addEdge, clearSolution],
  );

  // Запуск решения
  const handleSolve = () => {
    if (nodes.length < 2) {
      toast.error("Добавьте минимум 2 узла");
      return;
    }
    if (!hasSource || !hasSink) {
      toast.error("Добавьте источник и сток");
      return;
    }
    if (edges.length === 0) {
      toast.error("Добавьте рёбра между узлами");
      return;
    }
    solveMutation.mutate();
  };

  // Экспорт
  const handleExport = () => {
    try {
      const graph = getGraph();
      const json = JSON.stringify(
        graph,
        (_, value) => (typeof value === "bigint" ? value.toString() : value),
        2,
      );
      const blob = new Blob([json], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${name.replace(/\s+/g, "_")}.json`;
      a.click();
      URL.revokeObjectURL(url);
      toast.success("Граф экспортирован");
    } catch {
      toast.error("Ошибка экспорта");
    }
  };

  // Импорт
  const handleImport = () => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".json";
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;
      try {
        const text = await file.text();
        const graph = JSON.parse(text, (key, value) => {
          if (
            ["id", "from", "to", "sourceId", "sinkId"].includes(key) &&
            typeof value === "string" &&
            /^\d+$/.test(value)
          ) {
            return BigInt(value);
          }
          return value;
        });
        loadGraph(graph);
        toast.success("Граф загружен");
        setIsSaved(false);
      } catch {
        toast.error("Ошибка загрузки файла");
      }
    };
    input.click();
  };

  // Создание примера - всегда добавляем стоимость
  const handleCreateExample = () => {
    clearGraph();
    const source = addNode({
      x: 1,
      y: 3,
      type: NodeType.SOURCE,
      name: "Источник",
    });
    const w1 = addNode({
      x: 3,
      y: 1,
      type: NodeType.WAREHOUSE,
      name: "Склад А",
    });
    const w2 = addNode({
      x: 3,
      y: 5,
      type: NodeType.WAREHOUSE,
      name: "Склад Б",
    });
    const inter = addNode({
      x: 5,
      y: 3,
      type: NodeType.INTERSECTION,
      name: "Узел",
    });
    const d1 = addNode({
      x: 7,
      y: 2,
      type: NodeType.DELIVERY_POINT,
      name: "Точка 1",
    });
    const d2 = addNode({
      x: 7,
      y: 4,
      type: NodeType.DELIVERY_POINT,
      name: "Точка 2",
    });
    const sink = addNode({ x: 9, y: 3, type: NodeType.SINK, name: "Сток" });

    // Всегда добавляем стоимость - она просто не будет учитываться для других алгоритмов
    addEdge({ from: source.id, to: w1.id, capacity: 15, cost: 2 });
    addEdge({ from: source.id, to: w2.id, capacity: 12, cost: 3 });
    addEdge({ from: w1.id, to: inter.id, capacity: 10, cost: 1 });
    addEdge({ from: w2.id, to: inter.id, capacity: 8, cost: 2 });
    addEdge({ from: w1.id, to: d1.id, capacity: 7, cost: 4 });
    addEdge({ from: inter.id, to: d1.id, capacity: 5, cost: 1 });
    addEdge({ from: inter.id, to: d2.id, capacity: 6, cost: 2 });
    addEdge({ from: w2.id, to: d2.id, capacity: 9, cost: 3 });
    addEdge({ from: d1.id, to: sink.id, capacity: 12, cost: 1 });
    addEdge({ from: d2.id, to: sink.id, capacity: 14, cost: 1 });

    setSourceSink(source.id, sink.id);
    setName("Пример логистической сети");
    toast.success("Пример сети создан");
    setIsSaved(false);
  };

  const selectedNode =
    selectedNodeId !== null ? nodes.find((n) => n.id === selectedNodeId) : null;
  const selectedEdge = selectedEdgeKey
    ? edges.find(
        (e) => e.from === selectedEdgeKey.from && e.to === selectedEdgeKey.to,
      )
    : null;
  const canSolve =
    nodes.length >= 2 && edges.length > 0 && hasSource && hasSink;

  return (
    <div className="h-[calc(100vh-8rem)] flex gap-4">
      {/* Левая панель */}
      <div className="w-80 flex flex-col gap-4 overflow-y-auto">
        {/* Контролы */}
        <Card>
          <div className="flex items-center justify-between mb-4">
            <Input
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                setIsSaved(false);
              }}
              className="text-lg font-medium"
              placeholder="Название сети"
            />
            <button
              onClick={() => setShowSettings(!showSettings)}
              className={clsx(
                "p-2 rounded transition-colors",
                showSettings
                  ? "bg-primary-100 text-primary-600"
                  : "hover:bg-gray-100",
              )}
            >
              <Cog6ToothIcon className="w-5 h-5" />
            </button>
          </div>

          {/* Индикатор текущего алгоритма */}
          <div className="flex items-center gap-2 mb-4 text-sm">
            <span className="text-gray-500">Алгоритм:</span>
            <Badge variant={supportsCost ? "success" : "default"}>
              {algorithmName}
            </Badge>
            {supportsCost && (
              <CurrencyDollarIcon className="w-4 h-4 text-emerald-500" />
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            <Button
              onClick={handleSolve}
              loading={isLoading}
              disabled={!canSolve}
              icon={<PlayIcon className="w-4 h-4" />}
              className="flex-1"
            >
              Решить
            </Button>
            <Button
              variant={isSaved ? "ghost" : "secondary"}
              onClick={handleManualSave}
              loading={saveMutation.isPending}
              disabled={!hasSolution}
              icon={
                isSaved ? (
                  <CheckCircleIcon className="w-4 h-4 text-green-500" />
                ) : (
                  <BookmarkIcon className="w-4 h-4" />
                )
              }
              title={!hasSolution ? "Сначала выполните оптимизацию" : ""}
            >
              {isSaved ? "✓" : "Сохранить"}
            </Button>
          </div>

          {/* Автосохранение */}
          <label className="flex items-center gap-2 mt-3 text-sm">
            <input
              type="checkbox"
              checked={autoSave}
              onChange={(e) => setAutoSave(e.target.checked)}
              className="rounded text-primary-600"
            />
            <span className="text-gray-600">Автосохранение после решения</span>
          </label>

          {!hasSolution && nodes.length > 0 && (
            <p className="text-xs text-gray-400 mt-2">
              💡 Сохранение доступно после оптимизации
            </p>
          )}

          <div className="flex gap-2 mt-3">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleExport}
              disabled={!canSolve}
              className="flex-1"
            >
              <ArrowDownTrayIcon className="w-4 h-4 mr-1" />
              Экспорт
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleImport}
              className="flex-1"
            >
              <ArrowUpTrayIcon className="w-4 h-4 mr-1" />
              Импорт
            </Button>
          </div>

          <div className="flex gap-2 mt-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCreateExample}
              className="flex-1"
            >
              <ArrowPathIcon className="w-4 h-4 mr-1" />
              Пример
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={clearGraph}
              disabled={nodes.length === 0}
              className="flex-1 text-red-600 hover:bg-red-50"
            >
              <TrashIcon className="w-4 h-4 mr-1" />
              Очистить
            </Button>
          </div>

          {!canSolve && nodes.length > 0 && (
            <div className="mt-3 p-2 bg-yellow-50 border border-yellow-200 rounded text-xs text-yellow-800">
              {!hasSource && <p>⚠️ Добавьте источник</p>}
              {!hasSink && <p>⚠️ Добавьте сток</p>}
              {edges.length === 0 && <p>⚠️ Добавьте рёбра</p>}
            </div>
          )}
        </Card>

        {/* Настройки алгоритма */}
        {showSettings && (
          <AlgorithmSettings
            algorithm={algorithm}
            onAlgorithmChange={setAlgorithm}
          />
        )}

        {/* Палитра узлов */}
        <NodePalette
          onAddNode={handleAddNodeOfType}
          disabled={isLoading}
          hasSource={hasSource}
          hasSink={hasSink}
        />

        {/* Добавление ребра */}
        <Card>
          <Button
            variant="secondary"
            onClick={() => setShowAddEdge(true)}
            disabled={nodes.length < 2}
            className="w-full"
          >
            <PlusIcon className="w-4 h-4 mr-2" />
            Добавить ребро
          </Button>
          <p className="text-xs text-gray-400 mt-2">
            💡 Или соедините узлы на холсте
          </p>
        </Card>

        {/* Редактор узла */}
        {selectedNode && (
          <Card>
            <h3 className="font-medium mb-3">
              Узел #{String(selectedNode.id)}
            </h3>
            <Input
              label="Название"
              value={selectedNode.name ?? ""}
              onChange={(e) => {
                updateNode(selectedNode.id, { name: e.target.value });
                setIsSaved(false);
              }}
            />
            <Button
              variant="danger"
              onClick={() => removeNode(selectedNode.id)}
              className="w-full mt-4"
            >
              Удалить узел
            </Button>
          </Card>
        )}

        {/* Редактор ребра */}
        {selectedEdge && (
          <Card>
            <h3 className="font-medium mb-3">
              Ребро {String(selectedEdge.from)} → {String(selectedEdge.to)}
            </h3>
            <Input
              label="Пропускная способность"
              type="number"
              value={selectedEdge.capacity}
              onChange={(e) => {
                updateEdge(selectedEdge.from, selectedEdge.to, {
                  capacity: Number(e.target.value),
                });
                setIsSaved(false);
              }}
              min={0}
            />

            {/* Стоимость - показываем только для Min-Cost Flow, но данные всегда есть */}
            {supportsCost ? (
              <Input
                label="Стоимость"
                type="number"
                value={selectedEdge.cost ?? 0}
                onChange={(e) => {
                  updateEdge(selectedEdge.from, selectedEdge.to, {
                    cost: Number(e.target.value),
                  });
                  setIsSaved(false);
                }}
                min={0}
                className="mt-3"
              />
            ) : (
              <div className="mt-3 p-2 bg-gray-50 rounded text-xs text-gray-500 flex items-start gap-2">
                <InformationCircleIcon className="w-4 h-4 shrink-0 mt-0.5" />
                <span>
                  Стоимость ({selectedEdge.cost ?? 0}) сохранена, но не
                  учитывается текущим алгоритмом. Используйте Min-Cost Flow для
                  оптимизации затрат.
                </span>
              </div>
            )}

            <Button
              variant="danger"
              onClick={() => removeEdge(selectedEdge.from, selectedEdge.to)}
              className="w-full mt-4"
            >
              Удалить ребро
            </Button>
          </Card>
        )}

        {/* Результаты */}
        {flowResult && (
          <Card className="bg-green-50 border-green-200">
            <h3 className="font-medium text-green-800 mb-3">✅ Результат</h3>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-gray-600">Max Flow:</span>
                <span className="font-bold text-green-700 text-xl">
                  {flowResult.maxFlow}
                </span>
              </div>
              {/* Стоимость показываем только для Min-Cost Flow */}
              {supportsCost && flowResult.totalCost > 0 && (
                <div className="flex justify-between">
                  <span className="text-gray-600">Мин. стоимость:</span>
                  <span className="font-medium text-emerald-700">
                    ₽{flowResult.totalCost.toFixed(2)}
                  </span>
                </div>
              )}
              {metrics && (
                <div className="flex justify-between">
                  <span className="text-gray-600">Время:</span>
                  <span>{metrics.computationTimeMs.toFixed(2)} мс</span>
                </div>
              )}
            </div>
            {isSaved && (
              <div className="mt-3 pt-3 border-t border-green-200 text-sm text-green-600 flex items-center gap-1">
                <CheckCircleIcon className="w-4 h-4" />
                Сохранено в историю
              </div>
            )}
          </Card>
        )}

        {/* Статистика */}
        <Card className="bg-gray-50">
          <h3 className="font-medium mb-2 text-sm text-gray-600">Статистика</h3>
          <div className="grid grid-cols-2 gap-2 text-sm">
            <div>
              <span className="text-gray-500">Узлов:</span>{" "}
              <strong>{nodes.length}</strong>
            </div>
            <div>
              <span className="text-gray-500">Рёбер:</span>{" "}
              <strong>{edges.length}</strong>
            </div>
          </div>
        </Card>
      </div>

      {/* Canvas */}
      <div className="flex-1 card p-0 overflow-hidden relative">
        {nodes.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center bg-gray-50/80 z-10">
            <div className="text-center">
              <p className="text-gray-500 mb-4">Начните создавать сеть</p>
              <div className="flex gap-2 justify-center">
                <Button onClick={handleCreateExample}>Загрузить пример</Button>
                <Button
                  onClick={() => handleAddNodeOfType(NodeType.SOURCE)}
                  variant="secondary"
                >
                  Добавить источник
                </Button>
              </div>
            </div>
          </div>
        )}
        <GraphCanvas onNodeSelect={() => {}} onEdgeSelect={() => {}} />
      </div>

      <AddEdgeModal
        open={showAddEdge}
        onClose={() => setShowAddEdge(false)}
        nodes={nodes}
        onAdd={handleAddEdge}
        supportsCost={supportsCost}
      />
    </div>
  );
}
