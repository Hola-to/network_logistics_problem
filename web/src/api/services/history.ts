import { typedClient } from "../typedClient";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { Algorithm } from "@gen/logistics/common/v1/common_pb";
import type { Graph } from "@gen/logistics/common/v1/common_pb";
import type { SolveGraphResponse } from "@gen/logistics/gateway/v1/gateway_pb";

export const historyService = {
  saveCalculation: (params: {
    name: string;
    graph: Graph;
    result?: SolveGraphResponse;
    tags?: Record<string, string>;
  }) =>
    typedClient.saveCalculation({
      name: params.name,
      graph: params.graph,
      result: params.result,
      tags: params.tags,
    }),

  getCalculation: (calculationId: string) =>
    typedClient.getCalculation(calculationId),

  list: (params?: {
    limit?: number;
    offset?: number;
    algorithm?: Algorithm;
    tags?: string[];
    createdAfter?: Date;
    createdBefore?: Date;
    sortBy?: string;
    sortDesc?: boolean;
  }) =>
    typedClient.listCalculations({
      limit: params?.limit ?? 20,
      offset: params?.offset ?? 0,
      algorithm: params?.algorithm ?? Algorithm.UNSPECIFIED,
      tags: params?.tags ?? [],
      createdAfter: params?.createdAfter
        ? timestampFromDate(params.createdAfter)
        : undefined,
      createdBefore: params?.createdBefore
        ? timestampFromDate(params.createdBefore)
        : undefined,
      sortBy: params?.sortBy ?? "created_at",
      sortDesc: params?.sortDesc ?? true,
    }),

  deleteCalculation: (calculationId: string) =>
    typedClient.deleteCalculation(calculationId),

  getStatistics: (startTime?: Date, endTime?: Date) =>
    typedClient.getStatistics({
      startTime: startTime ? timestampFromDate(startTime) : undefined,
      endTime: endTime ? timestampFromDate(endTime) : undefined,
    }),
};
