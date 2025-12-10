import { useCallback, useMemo, useEffect } from "react";
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
import clsx from "clsx";

// ============================================================================
// Цвета для типов узлов
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
      {/* Входной handle (слева) */}
      <Handle
        type="target"
        position={Position.Left}
        className="w-3! h-3! bg-blue-500! border-2! border-white!"
      />

      {/* Содержимое узла */}
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

      {/* Выходной handle (справа) */}
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

  // Цвет в зависимости от загрузки
  const getEdgeColor = () => {
    if (utilization >= 1) return "#ef4444"; // red - перегружено
    if (utilization >= 0.9) return "#f97316"; // orange - почти полное
    if (utilization >= 0.5) return "#eab308"; // yellow - средняя загрузка
    if (flow > 0) return "#22c55e"; // green - есть поток
    return "#6b7280"; // gray - нет потока
  };

  const edgeColor = getEdgeColor();
  const strokeWidth = selected ? 4 : flow > 0 ? 3 : 2;

  return (
    <>
      {/* Основная линия ребра */}
      <path
        id={id}
        className="react-flow__edge-path"
        d={edgePath}
        strokeWidth={strokeWidth}
        stroke={edgeColor}
        fill="none"
        markerEnd={markerEnd}
        style={{
          strokeDasharray: flow === 0 ? "5,5" : undefined,
        }}
      />

      {/* Подсветка при выборе */}
      {selected && (
        <path
          d={edgePath}
          strokeWidth={strokeWidth + 4}
          stroke="#fbbf24"
          fill="none"
          opacity={0.3}
        />
      )}

      {/* Лейбл с информацией */}
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
            {/* Поток / Capacity */}
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

            {/* Стоимость */}
            {cost > 0 && (
              <div className="text-gray-500 text-center">
                ₽{cost.toFixed(1)}
              </div>
            )}

            {/* Индикатор загрузки */}
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

// ============================================================================
// Node & Edge Types
// ============================================================================

const nodeTypes: NodeTypes = {
  custom: CustomNode,
};

const edgeTypes: EdgeTypes = {
  custom: CustomEdge,
};

// ============================================================================
// GraphCanvas Component
// ============================================================================

interface GraphCanvasProps {
  onNodeSelect?: (nodeId: bigint | null) => void;
  onEdgeSelect?: (edge: { from: bigint; to: bigint } | null) => void;
  onNodeAdd?: (x: number, y: number) => void;
  readOnly?: boolean;
}

export default function GraphCanvas({
  onNodeSelect,
  onEdgeSelect,
  onNodeAdd,
  readOnly = false,
}: GraphCanvasProps) {
  const {
    nodes: graphNodes,
    edges: graphEdges,
    sourceId,
    sinkId,
    solvedGraph,
    addEdge: addGraphEdge,
    selectNode,
    selectEdge,
    updateNode,
  } = useGraphStore();

  // Определяем какие данные использовать
  const displayNodes = solvedGraph?.nodes ?? graphNodes;
  const displayEdges = solvedGraph?.edges ?? graphEdges;

  // Конвертируем узлы в формат ReactFlow
  const rfNodes = useMemo<RFNode[]>(() => {
    return displayNodes.map((node) => ({
      id: String(node.id),
      type: "custom",
      position: { x: node.x * 120, y: node.y * 120 }, // Увеличенный масштаб
      data: {
        label: node.name || `Узел ${node.id}`,
        nodeType: node.type,
        supply: node.supply || 0,
        demand: node.demand || 0,
        isSource: node.id === sourceId,
        isSink: node.id === sinkId,
      },
      draggable: !readOnly,
    }));
  }, [displayNodes, sourceId, sinkId, readOnly]);

  // Конвертируем рёбра в формат ReactFlow
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

  // ReactFlow state
  const [nodes, setNodes, onNodesChange] = useNodesState(rfNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(rfEdges);

  // Синхронизация при изменении данных
  useEffect(() => {
    setNodes(rfNodes);
  }, [rfNodes, setNodes]);

  useEffect(() => {
    setEdges(rfEdges);
  }, [rfEdges, setEdges]);

  // Обработчик соединения узлов
  const onConnect = useCallback(
    (connection: Connection) => {
      if (readOnly || !connection.source || !connection.target) return;

      const from = BigInt(connection.source);
      const to = BigInt(connection.target);

      // Проверяем, что ребро не к самому себе
      if (from === to) return;

      // Добавляем ребро в store
      const edge = addGraphEdge({
        from,
        to,
        capacity: 10,
        cost: 1,
      });

      if (edge) {
        console.log("Edge added:", edge);
      }
    },
    [readOnly, addGraphEdge],
  );

  // Обработчик клика на узел
  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: RFNode) => {
      const nodeId = BigInt(node.id);
      selectNode(nodeId);
      onNodeSelect?.(nodeId);
    },
    [selectNode, onNodeSelect],
  );

  // Обработчик клика на ребро
  const onEdgeClick = useCallback(
    (_: React.MouseEvent, edge: RFEdge) => {
      // Извлекаем ID из формата "e{from}-{to}"
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

  // Обработчик клика на пустое место
  const onPaneClick = useCallback(() => {
    selectNode(null);
    selectEdge(null);
    onNodeSelect?.(null);
    onEdgeSelect?.(null);
  }, [selectNode, selectEdge, onNodeSelect, onEdgeSelect]);

  // Обработчик двойного клика для добавления узла
  const onDoubleClick = useCallback(
    (event: React.MouseEvent) => {
      if (readOnly) return;

      // Получаем позицию относительно viewport
      const target = event.currentTarget as HTMLElement;
      const bounds = target.getBoundingClientRect();

      // Конвертируем в координаты графа
      const x = (event.clientX - bounds.left) / 120;
      const y = (event.clientY - bounds.top) / 120;

      onNodeAdd?.(x, y);
    },
    [readOnly, onNodeAdd],
  );

  // Обработчик перемещения узла
  const onNodeDragStop = useCallback(
    (_: React.MouseEvent, node: RFNode) => {
      if (readOnly) return;

      const nodeId = BigInt(node.id);
      updateNode(nodeId, {
        x: node.position.x / 120,
        y: node.position.y / 120,
      });
    },
    [readOnly, updateNode],
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
        onEdgeClick={onEdgeClick}
        onPaneClick={onPaneClick}
        onDoubleClick={onDoubleClick}
        onNodeDragStop={onNodeDragStop}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        connectionMode={ConnectionMode.Loose}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        defaultEdgeOptions={{
          type: "custom",
          markerEnd: {
            type: MarkerType.ArrowClosed,
            width: 15,
            height: 15,
          },
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
          🖱️ <strong>Двойной клик</strong> — добавить узел
        </p>
        <p>
          🔗 <strong>Перетащите</strong> от ● к ● — создать ребро
        </p>
        <p>
          📍 <strong>Клик</strong> на элемент — выбрать для редактирования
        </p>
      </div>
    </div>
  );
}
