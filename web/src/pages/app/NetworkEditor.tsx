import { useState, useCallback } from "react";
import { useMutation } from "@tanstack/react-query";
import toast from "react-hot-toast";
import {
  PlayIcon,
  TrashIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  Cog6ToothIcon,
  PlusIcon,
  ArrowPathIcon,
} from "@heroicons/react/24/outline";
import GraphCanvas from "@/components/visual/GraphCanvas";
import Card from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";
import Select from "@/components/ui/Select";
import Modal from "@/components/ui/Modal";
import { useGraphStore } from "@/stores/graphStore";
import { solverService, historyService } from "@/api/services";
import { NodeType, Algorithm } from "@gen/logistics/common/v1/common_pb";
import clsx from "clsx";
import type {
  SolveGraphResponse,
  SaveCalculationResponse,
} from "@gen/logistics/gateway/v1/gateway_pb";

// ============================================================================
// Конфигурация типов узлов
// ============================================================================

const NODE_TYPES_CONFIG = [
  {
    type: NodeType.SOURCE,
    label: "Источник",
    icon: "🟢",
    color: "bg-green-500",
    description: "Начальная точка потока",
  },
  {
    type: NodeType.SINK,
    label: "Сток",
    icon: "🔴",
    color: "bg-red-500",
    description: "Конечная точка потока",
  },
  {
    type: NodeType.WAREHOUSE,
    label: "Склад",
    icon: "📦",
    color: "bg-blue-500",
    description: "Промежуточное хранилище",
  },
  {
    type: NodeType.DELIVERY_POINT,
    label: "Точка доставки",
    icon: "📍",
    color: "bg-orange-500",
    description: "Пункт назначения",
  },
  {
    type: NodeType.INTERSECTION,
    label: "Перекрёсток",
    icon: "⚫",
    color: "bg-gray-500",
    description: "Транзитная точка",
  },
];

const ALGORITHMS = [
  { value: Algorithm.DINIC, label: "Dinic (рекомендуется)" },
  { value: Algorithm.EDMONDS_KARP, label: "Edmonds-Karp" },
  { value: Algorithm.PUSH_RELABEL, label: "Push-Relabel" },
  { value: Algorithm.MIN_COST, label: "Min-Cost Flow" },
  { value: Algorithm.FORD_FULKERSON, label: "Ford-Fulkerson" },
];

// ============================================================================
// Компонент палитры узлов
// ============================================================================

interface NodePaletteProps {
  onAddNode: (type: NodeType) => void;
  disabled?: boolean;
}

function NodePalette({ onAddNode, disabled }: NodePaletteProps) {
  return (
    <Card>
      <h3 className="font-medium mb-3 flex items-center gap-2">
        <PlusIcon className="w-4 h-4" />
        Добавить узел
      </h3>
      <div className="grid grid-cols-1 gap-2">
        {NODE_TYPES_CONFIG.map((config) => (
          <button
            key={config.type}
            onClick={() => onAddNode(config.type)}
            disabled={disabled}
            className={clsx(
              "flex items-center gap-3 p-3 rounded-lg border-2 border-dashed transition-all text-left",
              "hover:border-primary-400 hover:bg-primary-50",
              "disabled:opacity-50 disabled:cursor-not-allowed",
              "border-gray-200 bg-white",
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
              <p className="font-medium text-gray-900">{config.label}</p>
              <p className="text-xs text-gray-500 truncate">
                {config.description}
              </p>
            </div>
          </button>
        ))}
      </div>
      <p className="text-xs text-gray-400 mt-3">
        💡 Или дважды кликните на холст
      </p>
    </Card>
  );
}

// ============================================================================
// Компонент быстрого создания ребра
// ============================================================================

interface AddEdgeModalProps {
  open: boolean;
  onClose: () => void;
  nodes: Array<{ id: bigint; name?: string }>;
  onAdd: (from: bigint, to: bigint, capacity: number, cost: number) => void;
}

function AddEdgeModal({ open, onClose, nodes, onAdd }: AddEdgeModalProps) {
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

        <Input
          label="Стоимость за единицу"
          type="number"
          value={cost}
          onChange={(e) => setCost(Number(e.target.value))}
          min={0}
          step={0.1}
        />

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
// Главный компонент редактора
// ============================================================================

export default function NetworkEditor() {
  const {
    nodes,
    edges,
    sourceId,
    sinkId,
    name,
    algorithm,
    flowResult,
    metrics,
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
  } = useGraphStore();

  const [showSettings, setShowSettings] = useState(false);
  const [showAddEdge, setShowAddEdge] = useState(false);

  // Solve mutation
  const solveMutation = useMutation({
    mutationFn: () => {
      if (sourceId === null || sinkId === null) {
        return Promise.reject(new Error("Укажите источник и сток"));
      }
      const graph = getGraph();
      return solverService.solve({
        graph,
        algorithm,
        options: { returnPaths: true },
      });
    },
    onMutate: () => setLoading(true),
    onSuccess: (response: SolveGraphResponse) => {
      if (response.success && response.result && response.solvedGraph) {
        setSolution(
          response.solvedGraph,
          response.result,
          response.metrics ?? null,
        );
        toast.success(`Найден максимальный поток: ${response.result.maxFlow}`);
      } else {
        toast.error(response.errorMessage || "Ошибка решения");
      }
    },
    onError: (error: Error) => toast.error(error.message),
    onSettled: () => setLoading(false),
  });

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: async () => {
      const graph = getGraph();
      return historyService.saveCalculation({
        name,
        graph,
        result: flowResult
          ? {
              $typeName: "logistics.gateway.v1.SolveGraphResponse",
              success: true,
              result: flowResult,
              solvedGraph: getGraph(),
              metrics: metrics ?? undefined,
              errorMessage: "",
            }
          : undefined,
      });
    },
    onSuccess: (response: SaveCalculationResponse) => {
      toast.success(`Сохранено: ${response.calculationId}`);
    },
    onError: (error: Error) => toast.error(error.message),
  });

  // Добавление узла определённого типа
  const handleAddNodeOfType = useCallback(
    (type: NodeType) => {
      // Размещаем в центре с небольшим смещением
      const offsetX = (nodes.length % 5) * 1.5;
      const offsetY = Math.floor(nodes.length / 5) * 1.5;

      const newNode = addNode({
        x: 2 + offsetX,
        y: 2 + offsetY,
        type,
        name: `${NODE_TYPES_CONFIG.find((c) => c.type === type)?.label} ${nodes.length + 1}`,
      });

      // Автоматически устанавливаем source/sink
      if (type === NodeType.SOURCE && sourceId === null) {
        setSourceSink(newNode.id, sinkId);
        toast.success("Источник установлен");
      } else if (type === NodeType.SINK && sinkId === null) {
        setSourceSink(sourceId, newNode.id);
        toast.success("Сток установлен");
      }

      clearSolution();
    },
    [addNode, nodes.length, sourceId, sinkId, setSourceSink, clearSolution],
  );

  // Добавление узла на холсте
  const handleAddNodeOnCanvas = useCallback(
    (x: number, y: number) => {
      addNode({
        x,
        y,
        type: NodeType.INTERSECTION,
        name: `Узел ${nodes.length + 1}`,
      });
      clearSolution();
    },
    [addNode, nodes.length, clearSolution],
  );

  // Добавление ребра
  const handleAddEdge = useCallback(
    (from: bigint, to: bigint, capacity: number, cost: number) => {
      const edge = addEdge({ from, to, capacity, cost });
      if (edge) {
        toast.success("Ребро добавлено");
        clearSolution();
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
    if (sourceId === null || sinkId === null) {
      toast.error("Укажите источник и сток");
      return;
    }
    if (edges.length === 0) {
      toast.error("Добавьте рёбра между узлами");
      return;
    }
    solveMutation.mutate();
  };

  // Экспорт графа
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

  // Импорт графа
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
      } catch {
        toast.error("Ошибка загрузки файла");
      }
    };
    input.click();
  };

  // Создание примера сети
  const handleCreateExample = () => {
    clearGraph();

    // Добавляем узлы
    const source = addNode({
      x: 1,
      y: 3,
      type: NodeType.SOURCE,
      name: "Источник",
    });
    const warehouse1 = addNode({
      x: 3,
      y: 1,
      type: NodeType.WAREHOUSE,
      name: "Склад А",
    });
    const warehouse2 = addNode({
      x: 3,
      y: 5,
      type: NodeType.WAREHOUSE,
      name: "Склад Б",
    });
    const intersection = addNode({
      x: 5,
      y: 3,
      type: NodeType.INTERSECTION,
      name: "Узел",
    });
    const delivery1 = addNode({
      x: 7,
      y: 2,
      type: NodeType.DELIVERY_POINT,
      name: "Точка 1",
    });
    const delivery2 = addNode({
      x: 7,
      y: 4,
      type: NodeType.DELIVERY_POINT,
      name: "Точка 2",
    });
    const sink = addNode({ x: 9, y: 3, type: NodeType.SINK, name: "Сток" });

    // Добавляем рёбра
    addEdge({ from: source.id, to: warehouse1.id, capacity: 15, cost: 2 });
    addEdge({ from: source.id, to: warehouse2.id, capacity: 12, cost: 3 });
    addEdge({
      from: warehouse1.id,
      to: intersection.id,
      capacity: 10,
      cost: 1,
    });
    addEdge({ from: warehouse2.id, to: intersection.id, capacity: 8, cost: 2 });
    addEdge({ from: warehouse1.id, to: delivery1.id, capacity: 7, cost: 4 });
    addEdge({ from: intersection.id, to: delivery1.id, capacity: 5, cost: 1 });
    addEdge({ from: intersection.id, to: delivery2.id, capacity: 6, cost: 2 });
    addEdge({ from: warehouse2.id, to: delivery2.id, capacity: 9, cost: 3 });
    addEdge({ from: delivery1.id, to: sink.id, capacity: 12, cost: 1 });
    addEdge({ from: delivery2.id, to: sink.id, capacity: 14, cost: 1 });

    // Устанавливаем source/sink
    setSourceSink(source.id, sink.id);
    setName("Пример логистической сети");

    toast.success("Пример сети создан");
  };

  // Получаем выбранные элементы
  const selectedNode =
    selectedNodeId !== null ? nodes.find((n) => n.id === selectedNodeId) : null;

  const selectedEdge = selectedEdgeKey
    ? edges.find(
        (e) => e.from === selectedEdgeKey.from && e.to === selectedEdgeKey.to,
      )
    : null;

  // Проверка готовности к решению
  const canSolve =
    nodes.length >= 2 &&
    edges.length > 0 &&
    sourceId !== null &&
    sinkId !== null;

  return (
    <div className="h-[calc(100vh-8rem)] flex gap-4">
      {/* Левая панель */}
      <div className="w-80 flex flex-col gap-4 overflow-y-auto">
        {/* Заголовок и настройки */}
        <Card>
          <div className="flex items-center justify-between mb-4">
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
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

          {/* Основные кнопки */}
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
              variant="secondary"
              onClick={() => saveMutation.mutate()}
              loading={saveMutation.isPending}
              disabled={nodes.length === 0}
            >
              Сохранить
            </Button>
          </div>

          {/* Дополнительные кнопки */}
          <div className="flex gap-2 mt-3">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleExport}
              disabled={nodes.length === 0}
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

          {/* Предупреждения */}
          {!canSolve && nodes.length > 0 && (
            <div className="mt-3 p-2 bg-yellow-50 border border-yellow-200 rounded text-xs text-yellow-800">
              {sourceId === null && <p>⚠️ Добавьте источник</p>}
              {sinkId === null && <p>⚠️ Добавьте сток</p>}
              {edges.length === 0 && <p>⚠️ Добавьте рёбра</p>}
            </div>
          )}
        </Card>

        {/* Настройки алгоритма */}
        {showSettings && (
          <Card>
            <h3 className="font-medium mb-3">Настройки</h3>

            <Select
              label="Алгоритм"
              value={algorithm}
              onChange={(e) =>
                setAlgorithm(Number(e.target.value) as Algorithm)
              }
              options={ALGORITHMS.map((a) => ({
                value: a.value,
                label: a.label,
              }))}
            />

            <div className="grid grid-cols-2 gap-3 mt-3">
              <div>
                <label className="label">Источник</label>
                <select
                  value={sourceId?.toString() ?? ""}
                  onChange={(e) =>
                    setSourceSink(
                      e.target.value ? BigInt(e.target.value) : null,
                      sinkId,
                    )
                  }
                  className="input text-sm"
                >
                  <option value="">Не выбран</option>
                  {nodes.map((n) => (
                    <option key={String(n.id)} value={String(n.id)}>
                      {n.name || `Узел ${n.id}`}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="label">Сток</label>
                <select
                  value={sinkId?.toString() ?? ""}
                  onChange={(e) =>
                    setSourceSink(
                      sourceId,
                      e.target.value ? BigInt(e.target.value) : null,
                    )
                  }
                  className="input text-sm"
                >
                  <option value="">Не выбран</option>
                  {nodes.map((n) => (
                    <option key={String(n.id)} value={String(n.id)}>
                      {n.name || `Узел ${n.id}`}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </Card>
        )}

        {/* Палитра узлов */}
        <NodePalette onAddNode={handleAddNodeOfType} disabled={isLoading} />

        {/* Кнопка добавления ребра */}
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

        {/* Редактор выбранного узла */}
        {selectedNode && (
          <Card>
            <h3 className="font-medium mb-3">
              Редактирование узла #{String(selectedNode.id)}
            </h3>

            <Input
              label="Название"
              value={selectedNode.name ?? ""}
              onChange={(e) =>
                updateNode(selectedNode.id, { name: e.target.value })
              }
            />

            <div className="mt-3">
              <Select
                label="Тип"
                value={selectedNode.type}
                onChange={(e) =>
                  updateNode(selectedNode.id, {
                    type: Number(e.target.value) as NodeType,
                  })
                }
                options={NODE_TYPES_CONFIG.map((t) => ({
                  value: t.type,
                  label: `${t.icon} ${t.label}`,
                }))}
              />
            </div>

            <div className="flex gap-2 mt-4">
              {sourceId !== selectedNode.id && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSourceSink(selectedNode.id, sinkId)}
                  className="flex-1"
                >
                  Сделать источником
                </Button>
              )}
              {sinkId !== selectedNode.id && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSourceSink(sourceId, selectedNode.id)}
                  className="flex-1"
                >
                  Сделать стоком
                </Button>
              )}
            </div>

            <Button
              variant="danger"
              onClick={() => removeNode(selectedNode.id)}
              className="w-full mt-3"
            >
              Удалить узел
            </Button>
          </Card>
        )}

        {/* Редактор выбранного ребра */}
        {selectedEdge && (
          <Card>
            <h3 className="font-medium mb-3">
              Ребро {String(selectedEdge.from)} → {String(selectedEdge.to)}
            </h3>

            <Input
              label="Пропускная способность"
              type="number"
              value={selectedEdge.capacity}
              onChange={(e) =>
                updateEdge(selectedEdge.from, selectedEdge.to, {
                  capacity: Number(e.target.value),
                })
              }
              min={0}
            />

            <div className="mt-3">
              <Input
                label="Стоимость"
                type="number"
                value={selectedEdge.cost ?? 0}
                onChange={(e) =>
                  updateEdge(selectedEdge.from, selectedEdge.to, {
                    cost: Number(e.target.value),
                  })
                }
                min={0}
                step={0.1}
              />
            </div>

            {selectedEdge.currentFlow !== undefined &&
              selectedEdge.currentFlow > 0 && (
                <div className="mt-3 p-2 bg-blue-50 rounded text-sm">
                  <p>
                    Текущий поток: <strong>{selectedEdge.currentFlow}</strong> /{" "}
                    {selectedEdge.capacity}
                  </p>
                  <p className="text-xs text-gray-500">
                    Загрузка:{" "}
                    {(
                      (selectedEdge.currentFlow / selectedEdge.capacity) *
                      100
                    ).toFixed(1)}
                    %
                  </p>
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
            <h3 className="font-medium text-green-800 mb-3">
              ✅ Результат оптимизации
            </h3>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-gray-600">Max Flow:</span>
                <span className="font-bold text-green-700 text-xl">
                  {flowResult.maxFlow}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Общая стоимость:</span>
                <span className="font-medium">
                  ₽{flowResult.totalCost.toFixed(2)}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Итераций:</span>
                <span>{flowResult.iterations}</span>
              </div>
              {metrics && (
                <div className="flex justify-between">
                  <span className="text-gray-600">Время:</span>
                  <span>{metrics.computationTimeMs.toFixed(2)} мс</span>
                </div>
              )}
            </div>
          </Card>
        )}

        {/* Статистика графа */}
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
            <div>
              <span className="text-gray-500">Общая capacity:</span>{" "}
              <strong>{edges.reduce((sum, e) => sum + e.capacity, 0)}</strong>
            </div>
          </div>
        </Card>
      </div>

      {/* Холст */}
      <div className="flex-1 card p-0 overflow-hidden relative">
        {nodes.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center bg-gray-50/80 z-10">
            <div className="text-center">
              <p className="text-gray-500 mb-4">Начните создавать сеть</p>
              <div className="flex gap-2 justify-center">
                <Button onClick={handleCreateExample} variant="primary">
                  Загрузить пример
                </Button>
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

        <GraphCanvas
          onNodeSelect={() => {}}
          onEdgeSelect={() => {}}
          onNodeAdd={handleAddNodeOnCanvas}
        />
      </div>

      {/* Модальное окно добавления ребра */}
      <AddEdgeModal
        open={showAddEdge}
        onClose={() => setShowAddEdge(false)}
        nodes={nodes}
        onAdd={handleAddEdge}
      />
    </div>
  );
}
