import { typedClient } from "../typedClient";
import { Algorithm } from "@gen/logistics/common/v1/common_pb";
import type { Graph } from "@gen/logistics/common/v1/common_pb";
import type {
  CalculateLogisticsRequest,
  SolveOptions,
  BatchSolveItem,
} from "@gen/logistics/gateway/v1/gateway_pb";

export const solverService = {
  calculateLogistics: (request: Partial<CalculateLogisticsRequest>) =>
    typedClient.calculateLogistics({
      algorithm: Algorithm.DINIC,
      skipValidation: false,
      validationLevel: 2,
      calculateCost: true,
      findBottlenecks: true,
      calculateStatistics: true,
      ...request,
    } as unknown as CalculateLogisticsRequest),

  solve: (params: {
    graph: Graph;
    algorithm?: Algorithm;
    options?: Partial<SolveOptions>;
  }) =>
    typedClient.solveGraph({
      graph: params.graph,
      algorithm: params.algorithm ?? Algorithm.DINIC,
      options: {
        returnPaths: true,
        timeoutSeconds: 30,
        ...params.options,
      } as unknown as SolveOptions,
    }),

  /**
   * Streaming решение - возвращает async generator
   */
  solveStream: async function* (params: {
    graph: Graph;
    algorithm?: Algorithm;
    options?: Partial<SolveOptions>;
  }) {
    const stream = typedClient.solveGraphStream({
      graph: params.graph,
      algorithm: params.algorithm ?? Algorithm.DINIC,
      options: {
        returnPaths: true,
        timeoutSeconds: 60,
        ...params.options,
      } as unknown as SolveOptions,
    });

    yield* stream;
  },

  batchSolve: (
    items: Array<{ id: string; graph: Graph; algorithm?: Algorithm }>,
    parallel = true,
  ) =>
    typedClient.batchSolve({
      items: items.map(
        (item) =>
          ({
            id: item.id,
            graph: item.graph,
            algorithm: item.algorithm ?? Algorithm.DINIC,
          }) as unknown as BatchSolveItem,
      ),
      defaultAlgorithm: Algorithm.DINIC,
      parallel,
      maxConcurrent: 4,
    }),

  getAlgorithms: () => typedClient.getAlgorithms(),
  health: () => typedClient.health(),
  readiness: () => typedClient.readinessCheck(),
  info: () => typedClient.info(),
};
