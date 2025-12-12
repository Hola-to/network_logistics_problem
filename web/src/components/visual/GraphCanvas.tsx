import { useCallback, useMemo, useEffect, useState } from "react";
import ReactFlow, {
  Node as RFNode,
  Edge as RFEdge,
  Controls,
  Background,
  MiniMap,
  useNodesState,
  useEdgesState,
  Connection,
  NodeTypes,
  EdgeTypes,
  MarkerType,
  Handle,
  Position,
  getBezierPath,
  EdgeLabelRenderer,
  ConnectionMode,
  type NodeProps,
  type EdgeProps,
} from "reactflow";
import "reactflow/dist/style.css";
import { useGraphStore } from "@/stores/graphStore";
import { NodeType } from "@gen/logistics/common/v1/common_pb";
import Modal from "@/components/ui/Modal";
import Input from "@/components/ui/Input";
import Select from "@/components/ui/Select";
import Button from "@/components/ui/Button";
import clsx from "clsx";
import toast from "react-hot-toast";

// ============================================================================
// Конфигурация типов узлов
// ============================================================================

const NODE_STYLES: Record<
  number,
  { bg: string; border: string; text: string }
> = {
  [NodeType.UNSPECIFIED]: {
    bg: "bg-gray-100",
    border: "border-gray-400",
    text: "text-gray-700",
  },
  [NodeType.WAREHOUSE]: {
    bg: "bg-blue-100",
    border: "border-blue-500",
    text: "text-blue-800",
  },
  [NodeType.DELIVERY_POINT]: {
    bg: "bg-orange-100",
    border: "border-orange-500",
    text: "text-orange-800",
  },
  [NodeType.INTERSECTION]: {
    bg: "bg-gray-100",
    border: "border-gray-500",
    text: "text-gray-800",
  },
  [NodeType.SOURCE]: {
    bg: "bg-green-100",
    border: "border-green-500",
    text: "text-green-800",
  },
  [NodeType.SINK]: {
    bg: "bg-red-100",
    border: "border-red-500",
    text: "text-red-800",
  },
};

const NODE_ICONS: Record<number, string> = {
  [NodeType.UNSPECIFIED]: "⚫",
  [NodeType.WAREHOUSE]: "📦",
  [NodeType.DELIVERY_POINT]: "📍",
  [NodeType.INTERSECTION]: "⚫",
  [NodeType.SOURCE]: "🟢",
  [NodeType.SINK]: "🔴",
};

const NODE_TYPE_OPTIONS = [
  { value: NodeType.SOURCE, label: "🟢 Источник" },
  { value: NodeType.SINK, label: "🔴 Сток" },
  { value: NodeType.WAREHOUSE, label: "📦 Склад" },
  { value: NodeType.DELIVERY_POINT, label: "📍 Точка доставки" },
  { value: NodeType.INTERSECTION, label: "⚫ Перекрёсток" },
];

// ============================================================================
// Custom Node Component
// ============================================================================

function CustomNode({ data, selected }: NodeProps) {
  const nodeType = (data.nodeType as number) ?? NodeType.UNSPECIFIED;
  const styles = NODE_STYLES[nodeType] ?? NODE_STYLES[NodeType.UNSPECIFIED];
  const icon = NODE_ICONS[nodeType] ?? "⚫";
  const isSource = data.isSource as boolean;
  const isSink = data.isSink as boolean;

  return (
    <div
      className={clsx(
        "px-4 py-3 rounded-xl border-2 shadow-md transition-all min-w-[100px]",
        styles.bg,
        styles.border,
        selected && "ring-2 ring-yellow-400 ring-offset-2",
        (isSource || isSink) && "ring-2 ring-offset-1",
        isSource && "ring-green-400",
        isSink && "ring-red-400",
      )}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="w-3! h-3! bg-blue-500! border-2! border-white!"
      />

      <div className="flex items-center gap-2">
        <span className="text-lg">{icon}</span>
        <div className="flex-1 min-w-0">
          <div className={clsx("font-medium text-sm truncate", styles.text)}>
            {data.label as string}
          </div>
          {(isSource || isSink) && (
            <div className="text-xs opacity-70">
              {isSource ? "Источник" : "Сток"}
            </div>
          )}
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Right}
        className="w-3! h-3! bg-green-500! border-2! border-white!"
      />
    </div>
  );
}

// ============================================================================
// Custom Edge Component
// ============================================================================

function CustomEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  selected,
  markerEnd,
}: EdgeProps) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const flow = (data?.flow as number) ?? 0;
  const capacity = (data?.capacity as number) ?? 0;
  const cost = (data?.cost as number) ?? 0;
  const utilization = capacity > 0 ? flow / capacity : 0;

  const getEdgeColor = () => {
    if (utilization >= 1) return "#ef4444";
    if (utilization >= 0.9) return "#f97316";
    if (utilization >= 0.5) return "#eab308";
    if (flow > 0) return "#22c55e";
    return "#6b7280";
  };

  const edgeColor = getEdgeColor();
  const strokeWidth = selected ? 4 : flow > 0 ? 3 : 2;

  return (
    <>
      <path
        id={id}
        className="react-flow__edge-path"
        d={edgePath}
        strokeWidth={strokeWidth}
        stroke={edgeColor}
        fill="none"
        markerEnd={markerEnd}
        style={{ strokeDasharray: flow === 0 ? "5,5" : undefined }}
      />

      {selected && (
        <path
          d={edgePath}
          strokeWidth={strokeWidth + 4}
          stroke="#fbbf24"
          fill="none"
          opacity={0.3}
        />
      )}

      <EdgeLabelRenderer>
        <div
          style={{
            position: "absolute",
            transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
            pointerEvents: "all",
          }}
          className="nodrag nopan"
        >
          <div
            className={clsx(
              "px-2 py-1 rounded-lg text-xs font-medium shadow-sm border-2 bg-white",
              selected ? "border-yellow-400" : "border-gray-200",
              "hover:scale-105 transition-transform cursor-pointer",
            )}
          >
            <div className="flex items-center gap-1">
              <span
                className={clsx(
                  "font-bold",
                  flow > 0 ? "text-green-600" : "text-gray-400",
                )}
              >
                {flow}
              </span>
              <span className="text-gray-400">/</span>
              <span className="text-gray-700 font-semibold">{capacity}</span>
            </div>
            {cost > 0 && (
              <div className="text-gray-500 text-center">
                ₽{cost.toFixed(1)}
              </div>
            )}
            {capacity > 0 && (
              <div className="w-full h-1 bg-gray-200 rounded-full mt-1 overflow-hidden">
                <div
                  className={clsx(
                    "h-full rounded-full transition-all",
                    utilization >= 1
                      ? "bg-red-500"
                      : utilization >= 0.9
                        ? "bg-orange-500"
                        : utilization >= 0.5
                          ? "bg-yellow-500"
                          : "bg-green-500",
                  )}
                  style={{ width: `${Math.min(utilization * 100, 100)}%` }}
                />
              </div>
            )}
          </div>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

const nodeTypes: NodeTypes = { custom: CustomNode };
const edgeTypes: EdgeTypes = { custom: CustomEdge };

// ============================================================================
// Модальные окна редактирования
// ============================================================================

interface EditNodeModalProps {
  open: boolean;
  onClose: () => void;
  node: { id: bigint; name?: string; type: number } | null;
  sourceId: bigint | null;
  sinkId: bigint | null;
  onUpdate: (id: bigint, updates: { name?: string; type?: number }) => void;
  onDelete: (id: bigint) => void;
  onSetSource: (id: bigint) => void;
  onSetSink: (id: bigint) => void;
}

function EditNodeModal({
  open,
  onClose,
  node,
  sourceId,
  sinkId,
  onUpdate,
  onDelete,
  onSetSource,
  onSetSink,
}: EditNodeModalProps) {
  const [name, setName] = useState("");
  const [type, setType] = useState<number>(NodeType.INTERSECTION);

  useEffect(() => {
    if (node) {
      setName(node.name ?? "");
      setType(node.type);
    }
  }, [node]);

  if (!node) return null;

  const isSource = node.id === sourceId;
  const isSink = node.id === sinkId;
  const hasSource = sourceId !== null;
  const hasSink = sinkId !== null;

  const handleSave = () => {
    onUpdate(node.id, { name, type });
    onClose();
  };

  const handleSetSource = () => {
    if (hasSource && !isSource) {
      toast.error("Источник уже установлен. Удалите текущий источник.");
      return;
    }
    onSetSource(node.id);
    onClose();
  };

  const handleSetSink = () => {
    if (hasSink && !isSink) {
      toast.error("Сток уже установлен. Удалите текущий сток.");
      return;
    }
    onSetSink(node.id);
    onClose();
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`Редактировать узел #${String(node.id)}`}
      size="sm"
    >
      <div className="space-y-4">
        <Input
          label="Название"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />

        <Select
          label="Тип узла"
          value={type}
          onChange={(e) => setType(Number(e.target.value))}
          options={NODE_TYPE_OPTIONS.map((o) => ({
            value: o.value,
            label: o.label,
          }))}
        />

        {/* Source/Sink status */}
        <div className="flex gap-2">
          {isSource ? (
            <div className="flex-1 p-2 bg-green-50 border border-green-200 rounded text-center text-sm text-green-700">
              ✓ Источник
            </div>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleSetSource}
              disabled={hasSource && !isSource}
              className="flex-1"
            >
              Сделать источником
            </Button>
          )}

          {isSink ? (
            <div className="flex-1 p-2 bg-red-50 border border-red-200 rounded text-center text-sm text-red-700">
              ✓ Сток
            </div>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleSetSink}
              disabled={hasSink && !isSink}
              className="flex-1"
            >
              Сделать стоком
            </Button>
          )}
        </div>

        <div className="flex gap-2 pt-2 border-t">
          <Button onClick={handleSave} className="flex-1">
            Сохранить
          </Button>
          <Button
            variant="danger"
            onClick={() => {
              onDelete(node.id);
              onClose();
            }}
          >
            Удалить
          </Button>
        </div>
      </div>
    </Modal>
  );
}

interface EditEdgeModalProps {
  open: boolean;
  onClose: () => void;
  edge: {
    from: bigint;
    to: bigint;
    capacity: number;
    cost: number;
    currentFlow?: number;
  } | null;
  onUpdate: (
    from: bigint,
    to: bigint,
    updates: { capacity?: number; cost?: number },
  ) => void;
  onDelete: (from: bigint, to: bigint) => void;
}

function EditEdgeModal({
  open,
  onClose,
  edge,
  onUpdate,
  onDelete,
}: EditEdgeModalProps) {
  const [capacity, setCapacity] = useState(10);
  const [cost, setCost] = useState(1);

  useEffect(() => {
    if (edge) {
      setCapacity(edge.capacity);
      setCost(edge.cost);
    }
  }, [edge]);

  if (!edge) return null;

  const handleSave = () => {
    onUpdate(edge.from, edge.to, { capacity, cost });
    onClose();
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`Ребро ${String(edge.from)} → ${String(edge.to)}`}
      size="sm"
    >
      <div className="space-y-4">
        <Input
          label="Пропускная способность"
          type="number"
          value={capacity}
          onChange={(e) => setCapacity(Number(e.target.value))}
          min={1}
        />

        <Input
          label="Стоимость за единицу"
          type="number"
          value={cost}
          onChange={(e) => setCost(Number(e.target.value))}
          min={0}
          step={0.1}
        />

        {edge.currentFlow !== undefined && edge.currentFlow > 0 && (
          <div className="p-3 bg-blue-50 rounded-lg">
            <p className="text-sm text-blue-800">
              Текущий поток: <strong>{edge.currentFlow}</strong> /{" "}
              {edge.capacity}
            </p>
            <p className="text-xs text-blue-600">
              Загрузка: {((edge.currentFlow / edge.capacity) * 100).toFixed(1)}%
            </p>
          </div>
        )}

        <div className="flex gap-2 pt-2 border-t">
          <Button onClick={handleSave} className="flex-1">
            Сохранить
          </Button>
          <Button
            variant="danger"
            onClick={() => {
              onDelete(edge.from, edge.to);
              onClose();
            }}
          >
            Удалить
          </Button>
        </div>
      </div>
    </Modal>
  );
}

// ============================================================================
// GraphCanvas Component
// ============================================================================

interface GraphCanvasProps {
  onNodeSelect?: (nodeId: bigint | null) => void;
  onEdgeSelect?: (edge: { from: bigint; to: bigint } | null) => void;
  readOnly?: boolean;
}

export default function GraphCanvas({
  onNodeSelect,
  onEdgeSelect,
  readOnly = false,
}: GraphCanvasProps) {
  const {
    nodes: graphNodes,
    edges: graphEdges,
    sourceId,
    sinkId,
    solvedGraph,
    addEdge: addGraphEdge,
    updateNode,
    updateEdge,
    removeNode,
    removeEdge,
    selectNode,
    selectEdge,
    setSourceSink,
  } = useGraphStore();

  // Модальные окна
  const [editingNode, setEditingNode] = useState<{
    id: bigint;
    name?: string;
    type: number;
  } | null>(null);

  const [editingEdge, setEditingEdge] = useState<{
    from: bigint;
    to: bigint;
    capacity: number;
    cost: number;
    currentFlow?: number;
  } | null>(null);

  const displayNodes = solvedGraph?.nodes ?? graphNodes;
  const displayEdges = solvedGraph?.edges ?? graphEdges;

  // Конвертация в ReactFlow формат
  const rfNodes = useMemo<RFNode[]>(() => {
    return displayNodes.map((node) => ({
      id: String(node.id),
      type: "custom",
      position: { x: node.x * 120, y: node.y * 120 },
      data: {
        label: node.name || `Узел ${node.id}`,
        nodeType: node.type,
        isSource: node.id === sourceId,
        isSink: node.id === sinkId,
      },
      draggable: !readOnly,
    }));
  }, [displayNodes, sourceId, sinkId, readOnly]);

  const rfEdges = useMemo<RFEdge[]>(() => {
    return displayEdges.map((edge) => ({
      id: `e${edge.from}-${edge.to}`,
      source: String(edge.from),
      target: String(edge.to),
      type: "custom",
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 15,
        height: 15,
        color: edge.currentFlow && edge.currentFlow > 0 ? "#22c55e" : "#6b7280",
      },
      data: {
        capacity: edge.capacity,
        cost: edge.cost || 0,
        flow: edge.currentFlow || 0,
      },
    }));
  }, [displayEdges]);

  const [nodes, setNodes, onNodesChange] = useNodesState(rfNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(rfEdges);

  useEffect(() => {
    setNodes(rfNodes);
  }, [rfNodes, setNodes]);

  useEffect(() => {
    setEdges(rfEdges);
  }, [rfEdges, setEdges]);

  // Соединение узлов
  const onConnect = useCallback(
    (connection: Connection) => {
      if (readOnly || !connection.source || !connection.target) return;

      const from = BigInt(connection.source);
      const to = BigInt(connection.target);

      if (from === to) return;

      const edge = addGraphEdge({ from, to, capacity: 10, cost: 1 });
      if (edge) {
        toast.success("Ребро добавлено");
      } else {
        toast.error("Ребро уже существует");
      }
    },
    [readOnly, addGraphEdge],
  );

  // Одинарный клик - выбор для левой панели
  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: RFNode) => {
      const nodeId = BigInt(node.id);
      selectNode(nodeId);
      onNodeSelect?.(nodeId);
    },
    [selectNode, onNodeSelect],
  );

  // Двойной клик на узел - открыть модалку редактирования
  const onNodeDoubleClick = useCallback(
    (_: React.MouseEvent, node: RFNode) => {
      if (readOnly) return;

      const graphNode = graphNodes.find((n) => String(n.id) === node.id);
      if (graphNode) {
        setEditingNode({
          id: graphNode.id,
          name: graphNode.name,
          type: graphNode.type,
        });
      }
    },
    [readOnly, graphNodes],
  );

  // Одинарный клик на ребро
  const onEdgeClick = useCallback(
    (_: React.MouseEvent, edge: RFEdge) => {
      const match = edge.id.match(/^e(\d+)-(\d+)$/);
      if (match) {
        const from = BigInt(match[1]);
        const to = BigInt(match[2]);
        selectEdge({ from, to });
        onEdgeSelect?.({ from, to });
      }
    },
    [selectEdge, onEdgeSelect],
  );

  // Двойной клик на ребро - открыть модалку
  const onEdgeDoubleClick = useCallback(
    (_: React.MouseEvent, edge: RFEdge) => {
      if (readOnly) return;

      const match = edge.id.match(/^e(\d+)-(\d+)$/);
      if (match) {
        const from = BigInt(match[1]);
        const to = BigInt(match[2]);
        const graphEdge = graphEdges.find(
          (e) => e.from === from && e.to === to,
        );
        if (graphEdge) {
          setEditingEdge({
            from: graphEdge.from,
            to: graphEdge.to,
            capacity: graphEdge.capacity,
            cost: graphEdge.cost ?? 0,
            currentFlow: graphEdge.currentFlow,
          });
        }
      }
    },
    [readOnly, graphEdges],
  );

  // Клик на пустое место
  const onPaneClick = useCallback(() => {
    selectNode(null);
    selectEdge(null);
    onNodeSelect?.(null);
    onEdgeSelect?.(null);
  }, [selectNode, selectEdge, onNodeSelect, onEdgeSelect]);

  // Синхронизация позиций при перетаскивании
  const onNodeDragStop = useCallback(
    (_: React.MouseEvent, node: RFNode) => {
      if (readOnly) return;
      updateNode(BigInt(node.id), {
        x: node.position.x / 120,
        y: node.position.y / 120,
      });
    },
    [readOnly, updateNode],
  );

  // Обработчики модалок
  const handleUpdateNode = useCallback(
    (id: bigint, updates: { name?: string; type?: number }) => {
      updateNode(id, updates);
    },
    [updateNode],
  );

  const handleDeleteNode = useCallback(
    (id: bigint) => {
      removeNode(id);
      toast.success("Узел удалён");
    },
    [removeNode],
  );

  const handleSetSource = useCallback(
    (id: bigint) => {
      setSourceSink(id, sinkId);
      toast.success("Источник установлен");
    },
    [setSourceSink, sinkId],
  );

  const handleSetSink = useCallback(
    (id: bigint) => {
      setSourceSink(sourceId, id);
      toast.success("Сток установлен");
    },
    [setSourceSink, sourceId],
  );

  const handleUpdateEdge = useCallback(
    (
      from: bigint,
      to: bigint,
      updates: { capacity?: number; cost?: number },
    ) => {
      updateEdge(from, to, updates);
    },
    [updateEdge],
  );

  const handleDeleteEdge = useCallback(
    (from: bigint, to: bigint) => {
      removeEdge(from, to);
      toast.success("Ребро удалено");
    },
    [removeEdge],
  );

  return (
    <div className="w-full h-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={readOnly ? undefined : onNodesChange}
        onEdgesChange={readOnly ? undefined : onEdgesChange}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
        onNodeDoubleClick={onNodeDoubleClick}
        onEdgeClick={onEdgeClick}
        onEdgeDoubleClick={onEdgeDoubleClick}
        onPaneClick={onPaneClick}
        onNodeDragStop={onNodeDragStop}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        connectionMode={ConnectionMode.Loose}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        defaultEdgeOptions={{
          type: "custom",
          markerEnd: { type: MarkerType.ArrowClosed, width: 15, height: 15 },
        }}
        connectionLineStyle={{
          stroke: "#3b82f6",
          strokeWidth: 2,
          strokeDasharray: "5,5",
        }}
        snapToGrid
        snapGrid={[15, 15]}
        attributionPosition="bottom-right"
      >
        <Controls showInteractive={!readOnly} />
        <MiniMap
          nodeColor={(node) => {
            const nodeType = node.data?.nodeType as number;
            switch (nodeType) {
              case NodeType.SOURCE:
                return "#22c55e";
              case NodeType.SINK:
                return "#ef4444";
              case NodeType.WAREHOUSE:
                return "#3b82f6";
              case NodeType.DELIVERY_POINT:
                return "#f97316";
              default:
                return "#6b7280";
            }
          }}
          maskColor="rgba(0, 0, 0, 0.1)"
        />
        <Background gap={15} size={1} color="#e5e7eb" />
      </ReactFlow>

      {/* Подсказки */}
      <div className="absolute bottom-4 left-4 bg-white/90 backdrop-blur-sm rounded-lg shadow-sm px-3 py-2 text-xs text-gray-600 space-y-1">
        <p>
          🔗 <strong>Перетащите</strong> от ● к ● — создать ребро
        </p>
        <p>
          📍 <strong>Клик</strong> — выбрать элемент
        </p>
        <p>
          ✏️ <strong>Двойной клик</strong> — редактировать
        </p>
      </div>

      {/* Модальные окна */}
      <EditNodeModal
        open={editingNode !== null}
        onClose={() => setEditingNode(null)}
        node={editingNode}
        sourceId={sourceId}
        sinkId={sinkId}
        onUpdate={handleUpdateNode}
        onDelete={handleDeleteNode}
        onSetSource={handleSetSource}
        onSetSink={handleSetSink}
      />

      <EditEdgeModal
        open={editingEdge !== null}
        onClose={() => setEditingEdge(null)}
        edge={editingEdge}
        onUpdate={handleUpdateEdge}
        onDelete={handleDeleteEdge}
      />
    </div>
  );
}
