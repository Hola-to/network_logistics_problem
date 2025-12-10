import { typedClient } from "../typedClient";
import type { Graph } from "@gen/logistics/common/v1/common_pb";
import type {
  AnalysisOptions,
  CostOptions,
  ScenarioInput,
} from "@gen/logistics/gateway/v1/gateway_pb";

export const analyticsService = {
  analyzeGraph: (params: {
    graph: Graph;
    options?: Partial<AnalysisOptions>;
  }) =>
    typedClient.analyzeGraph({
      graph: params.graph,
      options: {
        analyzeCosts: true,
        findBottlenecks: true,
        calculateStatistics: true,
        suggestImprovements: true,
        bottleneckThreshold: 0.9,
        ...params.options,
      } as unknown as AnalysisOptions,
    }),

  calculateCost: (params: { graph: Graph; options?: Partial<CostOptions> }) =>
    typedClient.calculateCost({
      graph: params.graph,
      options: {
        currency: "RUB",
        includeFixedCosts: true,
        ...params.options,
      } as unknown as CostOptions,
    }),

  getBottlenecks: (graph: Graph, utilizationThreshold = 0.9, topN = 10) =>
    typedClient.getBottlenecks({ graph, utilizationThreshold, topN }),

  compareScenarios: (
    baseline: Graph,
    scenarios: Array<{ name: string; graph: Graph }>,
  ) =>
    typedClient.compareScenarios({
      baseline,
      scenarios: scenarios.map(
        (s) =>
          ({
            name: s.name,
            graph: s.graph,
          }) as unknown as ScenarioInput,
      ),
    }),
};
