import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { format } from "date-fns";
import { ru } from "date-fns/locale";
import toast from "react-hot-toast";
import { TrashIcon, EyeIcon, ArrowPathIcon } from "@heroicons/react/24/outline";
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
  ListCalculationsResponse,
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

  // Запрос списка расчётов
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["calculations"],
    queryFn: async () => {
      console.log("📡 Fetching calculations...");
      const response = await historyService.list({ limit: 50 });
      console.log("📥 Calculations response:", response);
      return response as ListCalculationsResponse;
    },
  });

  // Удаление
  const deleteMutation = useMutation({
    mutationFn: (id: string) => historyService.deleteCalculation(id),
    onSuccess: () => {
      toast.success("Расчёт удалён");
      queryClient.invalidateQueries({ queryKey: ["calculations"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  // Загрузка деталей
  const loadMutation = useMutation({
    mutationFn: (id: string) => historyService.getCalculation(id),
    onSuccess: (response: CalculationRecord) => {
      console.log("📥 Loaded calculation:", response);
      if (response.graph) {
        loadGraph(response.graph);
        toast.success("Граф загружен в редактор");
      } else {
        toast.error("Граф не найден в записи");
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

  // Получаем список расчётов
  const calculations = data?.calculations ?? [];
  const totalCount = data?.totalCount ?? 0n;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-gray-900">История расчётов</h1>
        <Card className="bg-red-50 border-red-200">
          <p className="text-red-800">
            Ошибка загрузки: {(error as Error).message}
          </p>
          <Button
            variant="secondary"
            onClick={() => refetch()}
            className="mt-4"
          >
            <ArrowPathIcon className="w-4 h-4 mr-2" />
            Повторить
          </Button>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">История расчётов</h1>
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => refetch()}>
            <ArrowPathIcon className="w-4 h-4 mr-1" />
            Обновить
          </Button>
          <p className="text-sm text-gray-500">Всего: {String(totalCount)}</p>
        </div>
      </div>

      {calculations.length === 0 ? (
        <Card className="text-center py-12">
          <div className="text-gray-400 text-5xl mb-4">📊</div>
          <p className="text-gray-500 text-lg">Нет сохранённых расчётов</p>
          <p className="text-sm text-gray-400 mt-2">
            Создайте граф в редакторе сети, запустите оптимизацию и нажмите
            "Сохранить"
          </p>
          <div className="mt-6">
            <a href="/network">
              <Button>Перейти в редактор</Button>
            </a>
          </div>
        </Card>
      ) : (
        <div className="space-y-4">
          {calculations.map((calc: CalculationSummary) => (
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
                  <div className="flex flex-wrap gap-4 mt-2 text-sm">
                    <span>
                      <span className="text-gray-500">Max Flow:</span>{" "}
                      <span className="font-medium text-primary-600">
                        {calc.maxFlow}
                      </span>
                    </span>
                    <span>
                      <span className="text-gray-500">Cost:</span>{" "}
                      <span className="font-medium">
                        ₽{calc.totalCost?.toFixed(2) ?? 0}
                      </span>
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
                      {calc.computationTimeMs?.toFixed(1) ?? 0} мс
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
                    title="Подробнее"
                  >
                    <EyeIcon className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => deleteMutation.mutate(calc.calculationId)}
                    loading={deleteMutation.isPending}
                    title="Удалить"
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
                <p className="text-sm text-gray-500">ID</p>
                <p className="font-mono text-sm">
                  {selectedCalc.calculationId}
                </p>
              </div>
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
                <p className="text-sm text-gray-500">Время вычисления</p>
                <p className="font-medium">
                  {selectedCalc.computationTimeMs?.toFixed(2) ?? 0} мс
                </p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4 p-4 bg-gray-50 rounded-lg">
              <div>
                <p className="text-sm text-gray-500">Максимальный поток</p>
                <p className="font-bold text-primary-600 text-2xl">
                  {selectedCalc.maxFlow}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Общая стоимость</p>
                <p className="font-bold text-2xl">
                  ₽{selectedCalc.totalCost?.toFixed(2) ?? 0}
                </p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-gray-500">Размер графа</p>
                <p className="font-medium">
                  {selectedCalc.nodeCount} узлов, {selectedCalc.edgeCount} рёбер
                </p>
              </div>
            </div>

            <div className="flex gap-2 pt-4 border-t">
              <Button
                onClick={() => handleLoadGraph(selectedCalc.calculationId)}
                loading={loadMutation.isPending}
                className="flex-1"
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
