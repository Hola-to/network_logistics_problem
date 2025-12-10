import { typedClient } from "../typedClient";
import { Algorithm } from "@gen/logistics/common/v1/common_pb";
import type { Graph } from "@gen/logistics/common/v1/common_pb";
import type {
  Modification,
  MonteCarloConfig,
  UncertaintySpec,
  SensitivityParameter,
  ResilienceConfig,
  FailureScenario,
  CriticalElementsConfig,
  WhatIfOptions,
} from "@gen/logistics/gateway/v1/gateway_pb";

export const simulationService = {
  runWhatIf: (params: {
    baselineGraph: Graph;
    modifications: Modification[];
    algorithm?: Algorithm;
    options?: Partial<WhatIfOptions>;
  }) =>
    typedClient.runWhatIf({
      baselineGraph: params.baselineGraph,
      modifications: params.modifications,
      algorithm: params.algorithm ?? Algorithm.DINIC,
      options: {
        compareWithBaseline: true,
        calculateCostImpact: true,
        findNewBottlenecks: true,
        returnModifiedGraph: true,
        ...params.options,
      } as unknown as WhatIfOptions,
    }),

  runMonteCarlo: (params: {
    graph: Graph;
    uncertainties: UncertaintySpec[];
    config?: Partial<MonteCarloConfig>;
    algorithm?: Algorithm;
  }) =>
    typedClient.runMonteCarlo({
      graph: params.graph,
      config: {
        numIterations: 1000,
        confidenceLevel: 0.95,
        parallel: true,
        randomSeed: 0n,
        ...params.config,
      } as unknown as MonteCarloConfig,
      uncertainties: params.uncertainties,
      algorithm: params.algorithm ?? Algorithm.DINIC,
    }),

  /**
   * Monte Carlo с потоковой передачей прогресса
   * Возвращает async generator
   */
  runMonteCarloStream: async function* (params: {
    graph: Graph;
    uncertainties: UncertaintySpec[];
    config?: Partial<MonteCarloConfig>;
    algorithm?: Algorithm;
  }) {
    const stream = typedClient.runMonteCarloStream({
      graph: params.graph,
      config: {
        numIterations: 1000,
        confidenceLevel: 0.95,
        parallel: true,
        randomSeed: 0n,
        ...params.config,
      } as unknown as MonteCarloConfig,
      uncertainties: params.uncertainties,
      algorithm: params.algorithm ?? Algorithm.DINIC,
    });

    yield* stream;
  },

  analyzeSensitivity: (params: {
    graph: Graph;
    parameters: SensitivityParameter[];
    algorithm?: Algorithm;
  }) =>
    typedClient.analyzeSensitivity({
      graph: params.graph,
      parameters: params.parameters,
      algorithm: params.algorithm ?? Algorithm.DINIC,
    }),

  analyzeResilience: (params: {
    graph: Graph;
    config?: Partial<ResilienceConfig>;
    algorithm?: Algorithm;
  }) =>
    typedClient.analyzeResilience({
      graph: params.graph,
      config: {
        maxFailuresToTest: 3,
        testCascadingFailures: true,
        loadFactor: 1.0,
        ...params.config,
      } as unknown as ResilienceConfig,
      algorithm: params.algorithm ?? Algorithm.DINIC,
    }),

  simulateFailures: (params: {
    graph: Graph;
    scenarios: FailureScenario[];
    algorithm?: Algorithm;
  }) =>
    typedClient.simulateFailures({
      graph: params.graph,
      scenarios: params.scenarios,
      algorithm: params.algorithm ?? Algorithm.DINIC,
    }),

  findCriticalElements: (params: {
    graph: Graph;
    config?: Partial<CriticalElementsConfig>;
    algorithm?: Algorithm;
  }) =>
    typedClient.findCriticalElements({
      graph: params.graph,
      config: {
        analyzeEdges: true,
        analyzeNodes: true,
        topN: 10,
        failureThreshold: 0.1,
        ...params.config,
      } as unknown as CriticalElementsConfig,
      algorithm: params.algorithm ?? Algorithm.DINIC,
    }),

  getSimulation: (simulationId: string) =>
    typedClient.getSimulation(simulationId),

  listSimulations: (limit = 20, offset = 0, simulationType?: string) =>
    typedClient.listSimulations({
      limit,
      offset,
      simulationType: simulationType ?? "",
    }),

  deleteSimulation: (simulationId: string) =>
    typedClient.deleteSimulation(simulationId),
};
