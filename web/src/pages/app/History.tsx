import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { format } from "date-fns";
import { ru } from "date-fns/locale";
import toast from "react-hot-toast";
import { TrashIcon, EyeIcon } from "@heroicons/react/24/outline";
import Card from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Spinner from "@/components/ui/Spinner";
import Badge from "@/components/ui/Badge";
import Modal from "@/components/ui/Modal";
import { historyService } from "@/api/services";
import { useGraphStore } from "@/stores/graphStore";
import { Algorithm } from "@gen/logistics/common/v1/common_pb";
import type {
  CalculationSummary,
  CalculationRecord,
} from "@gen/logistics/gateway/v1/gateway_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

const ALGORITHM_NAMES: Record<number, string> = {
  [Algorithm.UNSPECIFIED]: "Не указан",
  [Algorithm.EDMONDS_KARP]: "Edmonds-Karp",
  [Algorithm.DINIC]: "Dinic",
  [Algorithm.MIN_COST]: "Min-Cost",
  [Algorithm.PUSH_RELABEL]: "Push-Relabel",
  [Algorithm.FORD_FULKERSON]: "Ford-Fulkerson",
};

export default function History() {
  const queryClient = useQueryClient();
  const { loadGraph } = useGraphStore();
  const [selectedCalc, setSelectedCalc] = useState<CalculationSummary | null>(
    null,
  );
  const [detailsOpen, setDetailsOpen] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["calculations"],
    queryFn: () => historyService.list({ limit: 50 }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => historyService.deleteCalculation(id),
    onSuccess: () => {
      toast.success("Расчёт удалён");
      queryClient.invalidateQueries({ queryKey: ["calculations"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const loadMutation = useMutation({
    mutationFn: (id: string) => historyService.getCalculation(id),
    onSuccess: (response: CalculationRecord) => {
      if (response.graph) {
        loadGraph(response.graph);
        toast.success("Граф загружен");
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const handleViewDetails = (calc: CalculationSummary) => {
    setSelectedCalc(calc);
    setDetailsOpen(true);
  };

  const handleLoadGraph = (id: string) => {
    loadMutation.mutate(id);
    setDetailsOpen(false);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">История расчётов</h1>
        <p className="text-sm text-gray-500">
          Всего: {data?.totalCount?.toString() ?? 0}
        </p>
      </div>

      {!data?.calculations?.length ? (
        <Card className="text-center py-8">
          <p className="text-gray-500">Нет сохранённых расчётов</p>
          <p className="text-sm text-gray-400 mt-2">
            Создайте граф и запустите оптимизацию
          </p>
        </Card>
      ) : (
        <div className="space-y-4">
          {data.calculations.map((calc: CalculationSummary) => (
            <Card
              key={calc.calculationId}
              className="hover:shadow-md transition-shadow"
            >
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <h3 className="font-medium text-gray-900">
                      {calc.name || "Без названия"}
                    </h3>
                    <Badge variant="info">
                      {ALGORITHM_NAMES[calc.algorithm] ?? "Unknown"}
                    </Badge>
                  </div>
                  <p className="text-sm text-gray-500 mt-1">
                    {calc.createdAt
                      ? format(
                          timestampDate(calc.createdAt),
                          "dd MMMM yyyy, HH:mm",
                          { locale: ru },
                        )
                      : "—"}
                  </p>
                  <div className="flex gap-4 mt-2 text-sm">
                    <span>
                      <span className="text-gray-500">Max Flow:</span>{" "}
                      <span className="font-medium text-primary-600">
                        {calc.maxFlow}
                      </span>
                    </span>
                    <span>
                      <span className="text-gray-500">Cost:</span>{" "}
                      <span className="font-medium">{calc.totalCost}</span>
                    </span>
                    <span>
                      <span className="text-gray-500">Узлов:</span>{" "}
                      {calc.nodeCount}
                    </span>
                    <span>
                      <span className="text-gray-500">Рёбер:</span>{" "}
                      {calc.edgeCount}
                    </span>
                    <span>
                      <span className="text-gray-500">Время:</span>{" "}
                      {calc.computationTimeMs.toFixed(1)} мс
                    </span>
                  </div>
                  {calc.tags && calc.tags.length > 0 && (
                    <div className="flex gap-1 mt-2">
                      {calc.tags.map((tag: string) => (
                        <Badge key={tag} variant="default" size="sm">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  )}
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleViewDetails(calc)}
                  >
                    <EyeIcon className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => deleteMutation.mutate(calc.calculationId)}
                    loading={deleteMutation.isPending}
                  >
                    <TrashIcon className="w-4 h-4 text-red-500" />
                  </Button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Details Modal */}
      <Modal
        open={detailsOpen}
        onClose={() => setDetailsOpen(false)}
        title={selectedCalc?.name || "Детали расчёта"}
        size="lg"
      >
        {selectedCalc && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-gray-500">Алгоритм</p>
                <p className="font-medium">
                  {ALGORITHM_NAMES[selectedCalc.algorithm]}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Дата создания</p>
                <p className="font-medium">
                  {selectedCalc.createdAt
                    ? format(timestampDate(selectedCalc.createdAt), "PPpp", {
                        locale: ru,
                      })
                    : "—"}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Максимальный поток</p>
                <p className="font-medium text-primary-600 text-xl">
                  {selectedCalc.maxFlow}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Общая стоимость</p>
                <p className="font-medium text-xl">{selectedCalc.totalCost}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Размер графа</p>
                <p className="font-medium">
                  {selectedCalc.nodeCount} узлов, {selectedCalc.edgeCount} рёбер
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Время вычисления</p>
                <p className="font-medium">
                  {selectedCalc.computationTimeMs.toFixed(2)} мс
                </p>
              </div>
            </div>

            <div className="flex gap-2 pt-4 border-t">
              <Button
                onClick={() => handleLoadGraph(selectedCalc.calculationId)}
                loading={loadMutation.isPending}
              >
                Загрузить в редактор
              </Button>
              <Button variant="secondary" onClick={() => setDetailsOpen(false)}>
                Закрыть
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
