import { useMutation, useQuery } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { solverService, historyService } from "@/api/services";
import { useGraphStore } from "@/stores/graphStore";
import type { ErrorDetail } from "@gen/logistics/common/v1/common_pb";

export function useSolver() {
  const store = useGraphStore();

  const algorithmsQuery = useQuery({
    queryKey: ["algorithms"],
    queryFn: () => solverService.getAlgorithms(),
    staleTime: Infinity,
  });

  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: () => solverService.health(),
    refetchInterval: 30000,
  });

  const solveMutation = useMutation({
    mutationFn: async () => {
      const graph = store.getGraph();
      return solverService.solve({
        graph,
        algorithm: store.algorithm,
      });
    },
    onMutate: () => {
      store.setLoading(true);
      store.setError(null);
    },
    onSuccess: (response) => {
      if (response.success && response.result && response.solvedGraph) {
        store.setSolution(
          response.solvedGraph,
          response.result,
          response.metrics ?? null,
        );
        toast.success(`Найден максимальный поток: ${response.result.maxFlow}`);
      } else {
        store.setError(response.errorMessage || "Ошибка решения");
        toast.error(response.errorMessage || "Ошибка решения");
      }
    },
    onError: (error: Error) => {
      store.setError(error.message);
      toast.error(error.message);
    },
    onSettled: () => {
      store.setLoading(false);
    },
  });

  const calculateMutation = useMutation({
    mutationFn: async (options?: {
      saveToHistory?: boolean;
      name?: string;
    }) => {
      const graph = store.getGraph();
      return solverService.calculateLogistics({
        graph,
        algorithm: store.algorithm,
        saveToHistory: options?.saveToHistory ?? false,
        calculationName: options?.name ?? store.name,
      });
    },
    onMutate: () => {
      store.setLoading(true);
      store.setError(null);
    },
    onSuccess: (response) => {
      if (response.success && response.optimization) {
        store.setSolution(
          response.optimization.solvedGraph ?? null,
          response.optimization as any,
          null,
        );
        toast.success(`Max Flow: ${response.optimization.maxFlow}`);
      } else if (response.errors?.length) {
        const errorMsg = response.errors
          .map((e: ErrorDetail) => e.message)
          .join(", ");
        store.setError(errorMsg);
        toast.error(errorMsg);
      }
    },
    onError: (error: Error) => {
      store.setError(error.message);
      toast.error(error.message);
    },
    onSettled: () => {
      store.setLoading(false);
    },
  });

  const saveMutation = useMutation({
    mutationFn: async (name?: string) => {
      const graph = store.getGraph();
      return historyService.saveCalculation({
        name: name ?? store.name,
        graph,
        result: store.flowResult
          ? {
              $typeName: "logistics.gateway.v1.SolveGraphResponse",
              success: true,
              result: store.flowResult,
              solvedGraph: store.solvedGraph ?? undefined,
              metrics: store.metrics ?? undefined,
              errorMessage: "",
            }
          : undefined,
      });
    },
    onSuccess: (response) => {
      toast.success(`Сохранено: ${response.calculationId}`);
    },
    onError: (error: Error) => {
      toast.error(error.message);
    },
  });

  return {
    algorithms: algorithmsQuery.data?.algorithms ?? [],
    isHealthy: healthQuery.data?.status === "HEALTHY",
    services: healthQuery.data?.services,
    solve: solveMutation.mutate,
    calculate: calculateMutation.mutate,
    save: saveMutation.mutate,
    isSolving: solveMutation.isPending,
    isCalculating: calculateMutation.isPending,
    isSaving: saveMutation.isPending,
  };
}
