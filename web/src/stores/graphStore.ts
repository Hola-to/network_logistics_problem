import { create } from "@bufbuild/protobuf";
import { create as createStore } from "zustand";
import {
  Algorithm,
  NodeType,
  RoadType,
  FlowStatus,
  NodeSchema,
  EdgeSchema,
  GraphSchema,
} from "@gen/logistics/common/v1/common_pb";
import type {
  Node,
  Edge,
  Graph,
  FlowResult,
} from "@gen/logistics/common/v1/common_pb";
import type { SolveMetrics } from "@gen/logistics/gateway/v1/gateway_pb";

export { Algorithm, NodeType, RoadType, FlowStatus };

// ============================================================================
// Helpers
// ============================================================================

const toBigInt = (n: number | bigint): bigint =>
  typeof n === "bigint" ? n : BigInt(n);

const createNode = (
  data: Partial<Node> & { id: bigint; x: number; y: number },
): Node => {
  return create(NodeSchema, {
    id: data.id,
    x: data.x,
    y: data.y,
    type: data.type ?? NodeType.INTERSECTION,
    name: data.name ?? `Узел ${data.id}`,
    metadata: data.metadata ?? {},
    supply: data.supply ?? 0,
    demand: data.demand ?? 0,
  }) as unknown as Node;
};

const createEdge = (
  data: Partial<Edge> & { from: bigint; to: bigint },
): Edge => {
  return create(EdgeSchema, {
    from: data.from,
    to: data.to,
    capacity: data.capacity ?? 10,
    cost: data.cost ?? 1,
    length: data.length ?? 1,
    roadType: data.roadType ?? RoadType.PRIMARY,
    currentFlow: data.currentFlow ?? 0,
    bidirectional: data.bidirectional ?? false,
  }) as unknown as Edge;
};

const createGraph = (data: {
  nodes: Node[];
  edges: Edge[];
  sourceId: bigint;
  sinkId: bigint;
  name: string;
  metadata: Record<string, string>;
}): Graph => {
  return create(GraphSchema, {
    nodes: data.nodes,
    edges: data.edges,
    sourceId: data.sourceId,
    sinkId: data.sinkId,
    name: data.name,
    metadata: data.metadata,
  }) as unknown as Graph;
};

// ============================================================================
// Store Interface
// ============================================================================

interface GraphState {
  nodes: Node[];
  edges: Edge[];
  sourceId: bigint | null;
  sinkId: bigint | null;
  name: string;
  metadata: Record<string, string>;

  // Результаты решения (храним как plain data, не protobuf)
  solvedGraph: Graph | null;
  flowResult: FlowResult | null;
  metrics: SolveMetrics | null;

  // UI state
  selectedNodeId: bigint | null;
  selectedEdgeKey: { from: bigint; to: bigint } | null;
  algorithm: Algorithm;
  isLoading: boolean;
  error: string | null;

  // Node actions
  addNode: (node: Partial<Node> & { x: number; y: number }) => Node;
  updateNode: (id: bigint | number, updates: Partial<Node>) => void;
  removeNode: (id: bigint | number) => void;

  // Edge actions
  addEdge: (
    edge: Partial<Edge> & { from: bigint | number; to: bigint | number },
  ) => Edge | null;
  updateEdge: (
    from: bigint | number,
    to: bigint | number,
    updates: Partial<Edge>,
  ) => void;
  removeEdge: (from: bigint | number, to: bigint | number) => void;

  // Graph actions
  setSourceSink: (
    sourceId: bigint | number | null,
    sinkId: bigint | number | null,
  ) => void;
  setName: (name: string) => void;
  setMetadata: (metadata: Record<string, string>) => void;
  loadGraph: (graph: Graph) => void;
  getGraph: () => Graph;
  clearGraph: () => void;

  // Solution actions
  setSolution: (
    solvedGraph: Graph | null,
    flowResult: FlowResult | null,
    metrics: SolveMetrics | null,
  ) => void;
  clearSolution: () => void;
  hasSolution: () => boolean;

  // UI actions
  selectNode: (id: bigint | number | null) => void;
  selectEdge: (
    key: { from: bigint | number; to: bigint | number } | null,
  ) => void;
  setAlgorithm: (algorithm: Algorithm) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

// ============================================================================
// Store Implementation
// ============================================================================

let nodeIdCounter = 1n;

export const useGraphStore = createStore<GraphState>((set, get) => ({
  nodes: [],
  edges: [],
  sourceId: null,
  sinkId: null,
  name: "Новая сеть",
  metadata: {},
  solvedGraph: null,
  flowResult: null,
  metrics: null,
  selectedNodeId: null,
  selectedEdgeKey: null,
  algorithm: Algorithm.DINIC,
  isLoading: false,
  error: null,

  // ==========================================================================
  // Node Actions
  // ==========================================================================

  addNode: (nodeData) => {
    const id =
      nodeData.id !== undefined ? toBigInt(nodeData.id) : nodeIdCounter++;

    const node = createNode({
      ...nodeData,
      id,
      x: nodeData.x,
      y: nodeData.y,
    });

    set((state) => ({
      nodes: [...state.nodes, node],
      solvedGraph: null,
      flowResult: null,
      metrics: null,
    }));

    return node;
  },

  updateNode: (id, updates) => {
    const bigId = toBigInt(id);
    set((state) => ({
      nodes: state.nodes.map((n) => {
        if (n.id !== bigId) return n;
        return createNode({
          ...n,
          ...updates,
          id: n.id,
          x: updates.x ?? n.x,
          y: updates.y ?? n.y,
        });
      }),
      solvedGraph: null,
      flowResult: null,
      metrics: null,
    }));
  },

  removeNode: (id) => {
    const bigId = toBigInt(id);
    set((state) => ({
      nodes: state.nodes.filter((n) => n.id !== bigId),
      edges: state.edges.filter((e) => e.from !== bigId && e.to !== bigId),
      sourceId: state.sourceId === bigId ? null : state.sourceId,
      sinkId: state.sinkId === bigId ? null : state.sinkId,
      selectedNodeId:
        state.selectedNodeId === bigId ? null : state.selectedNodeId,
      solvedGraph: null,
      flowResult: null,
      metrics: null,
    }));
  },

  // ==========================================================================
  // Edge Actions
  // ==========================================================================

  addEdge: (edgeData) => {
    const from = toBigInt(edgeData.from);
    const to = toBigInt(edgeData.to);

    const state = get();
    const exists = state.edges.some((e) => e.from === from && e.to === to);
    if (exists) return null;

    const edge = createEdge({
      ...edgeData,
      from,
      to,
    });

    set((state) => ({
      edges: [...state.edges, edge],
      solvedGraph: null,
      flowResult: null,
      metrics: null,
    }));

    return edge;
  },

  updateEdge: (from, to, updates) => {
    const bigFrom = toBigInt(from);
    const bigTo = toBigInt(to);
    set((state) => ({
      edges: state.edges.map((e) => {
        if (e.from !== bigFrom || e.to !== bigTo) return e;
        return createEdge({
          ...e,
          ...updates,
          from: e.from,
          to: e.to,
        });
      }),
      solvedGraph: null,
      flowResult: null,
      metrics: null,
    }));
  },

  removeEdge: (from, to) => {
    const bigFrom = toBigInt(from);
    const bigTo = toBigInt(to);
    set((state) => ({
      edges: state.edges.filter((e) => !(e.from === bigFrom && e.to === bigTo)),
      selectedEdgeKey:
        state.selectedEdgeKey?.from === bigFrom &&
        state.selectedEdgeKey?.to === bigTo
          ? null
          : state.selectedEdgeKey,
      solvedGraph: null,
      flowResult: null,
      metrics: null,
    }));
  },

  // ==========================================================================
  // Graph Actions
  // ==========================================================================

  setSourceSink: (sourceId, sinkId) =>
    set({
      sourceId: sourceId !== null ? toBigInt(sourceId) : null,
      sinkId: sinkId !== null ? toBigInt(sinkId) : null,
    }),

  setName: (name) => set({ name }),

  setMetadata: (metadata) => set({ metadata }),

  loadGraph: (graph) => {
    const maxId = graph.nodes.reduce((max, n) => (n.id > max ? n.id : max), 0n);
    nodeIdCounter = maxId + 1n;

    const nodes = graph.nodes.map((n) =>
      createNode({
        id: n.id,
        x: n.x,
        y: n.y,
        type: n.type,
        name: n.name,
        metadata: { ...n.metadata },
        supply: n.supply,
        demand: n.demand,
      }),
    );

    const edges = graph.edges.map((e) =>
      createEdge({
        from: e.from,
        to: e.to,
        capacity: e.capacity,
        cost: e.cost,
        length: e.length,
        roadType: e.roadType,
        currentFlow: e.currentFlow,
        bidirectional: e.bidirectional,
      }),
    );

    set({
      nodes,
      edges,
      sourceId: graph.sourceId,
      sinkId: graph.sinkId,
      name: graph.name || "Загруженная сеть",
      metadata: { ...graph.metadata },
      solvedGraph: null,
      flowResult: null,
      metrics: null,
      selectedNodeId: null,
      selectedEdgeKey: null,
    });
  },

  getGraph: (): Graph => {
    const state = get();

    if (state.sourceId === null || state.sinkId === null) {
      throw new Error("Source and Sink must be set before getting graph");
    }

    return createGraph({
      nodes: state.nodes,
      edges: state.edges,
      sourceId: state.sourceId,
      sinkId: state.sinkId,
      name: state.name,
      metadata: state.metadata,
    });
  },

  clearGraph: () => {
    nodeIdCounter = 1n;
    set({
      nodes: [],
      edges: [],
      sourceId: null,
      sinkId: null,
      name: "Новая сеть",
      metadata: {},
      solvedGraph: null,
      flowResult: null,
      metrics: null,
      selectedNodeId: null,
      selectedEdgeKey: null,
      error: null,
    });
  },

  // ==========================================================================
  // Solution Actions
  // ==========================================================================

  setSolution: (solvedGraph, flowResult, metrics) => {
    set({
      solvedGraph,
      flowResult,
      metrics,
      error: null,
    });
  },

  clearSolution: () =>
    set({
      solvedGraph: null,
      flowResult: null,
      metrics: null,
    }),

  hasSolution: () => {
    const state = get();
    return state.flowResult !== null;
  },

  // ==========================================================================
  // UI Actions
  // ==========================================================================

  selectNode: (id) =>
    set({
      selectedNodeId: id !== null ? toBigInt(id) : null,
      selectedEdgeKey: null,
    }),

  selectEdge: (key) =>
    set({
      selectedEdgeKey: key
        ? { from: toBigInt(key.from), to: toBigInt(key.to) }
        : null,
      selectedNodeId: null,
    }),

  setAlgorithm: (algorithm) => set({ algorithm }),

  setLoading: (isLoading) => set({ isLoading }),

  setError: (error) => set({ error }),
}));
