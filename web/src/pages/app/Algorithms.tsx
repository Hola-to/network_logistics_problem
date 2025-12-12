import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  AcademicCapIcon,
  ClockIcon,
  CpuChipIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  ArrowRightIcon,
  SparklesIcon,
  BeakerIcon,
  BoltIcon,
  CurrencyDollarIcon,
  ScaleIcon,
} from "@heroicons/react/24/outline";
import { solverService } from "@/api/services";
import Card from "@/components/ui/Card";
import Button from "@/components/ui/Button";
import Spinner from "@/components/ui/Spinner";
import Badge from "@/components/ui/Badge";
import { Algorithm } from "@gen/logistics/common/v1/common_pb";
import type { AlgorithmInfo } from "@gen/logistics/gateway/v1/gateway_pb";
import clsx from "clsx";

// ============================================================================
// Конфигурация отображения алгоритмов
// ============================================================================

const ALGORITHM_DISPLAY: Record<
  number,
  {
    icon: React.ComponentType<{ className?: string }>;
    color: string;
    gradient: string;
    tagline: string;
  }
> = {
  [Algorithm.FORD_FULKERSON]: {
    icon: AcademicCapIcon,
    color: "text-purple-600",
    gradient: "from-purple-500 to-purple-600",
    tagline: "Классический алгоритм для обучения",
  },
  [Algorithm.EDMONDS_KARP]: {
    icon: BeakerIcon,
    color: "text-blue-600",
    gradient: "from-blue-500 to-blue-600",
    tagline: "Надёжный выбор для большинства задач",
  },
  [Algorithm.DINIC]: {
    icon: BoltIcon,
    color: "text-green-600",
    gradient: "from-green-500 to-green-600",
    tagline: "Рекомендуется для продакшена",
  },
  [Algorithm.PUSH_RELABEL]: {
    icon: CpuChipIcon,
    color: "text-orange-600",
    gradient: "from-orange-500 to-orange-600",
    tagline: "Для очень больших и плотных графов",
  },
  [Algorithm.MIN_COST]: {
    icon: CurrencyDollarIcon,
    color: "text-emerald-600",
    gradient: "from-emerald-500 to-emerald-600",
    tagline: "Когда важна минимизация затрат",
  },
};

const BEST_FOR_LABELS: Record<string, { label: string; icon: string }> = {
  small_graphs: { label: "Малые графы", icon: "📊" },
  integer_capacities: { label: "Целые пропускные способности", icon: "🔢" },
  educational: { label: "Обучение", icon: "📚" },
  general_graphs: { label: "Общие графы", icon: "🌐" },
  small_to_medium_size: { label: "Малые и средние графы", icon: "📈" },
  large_graphs: { label: "Большие графы", icon: "🏔️" },
  unit_capacity_graphs: {
    label: "Единичные пропускные способности",
    icon: "1️⃣",
  },
  bipartite_matching: { label: "Двудольное сопоставление", icon: "🔗" },
  dense_graphs: { label: "Плотные графы", icon: "🕸️" },
  very_large_graphs: { label: "Очень большие графы", icon: "🌌" },
  cost_optimization: { label: "Оптимизация стоимости", icon: "💰" },
  transportation_problems: { label: "Транспортные задачи", icon: "🚚" },
  assignment_problems: { label: "Задачи назначения", icon: "📋" },
};

// ============================================================================
// Компонент карточки алгоритма
// ============================================================================

interface AlgorithmCardProps {
  info: AlgorithmInfo;
  isRecommended?: boolean;
}

function AlgorithmCard({ info, isRecommended }: AlgorithmCardProps) {
  const display = ALGORITHM_DISPLAY[info.algorithm] ?? {
    icon: CpuChipIcon,
    color: "text-gray-600",
    gradient: "from-gray-500 to-gray-600",
    tagline: "",
  };

  const Icon = display.icon;

  return (
    <Card
      className={clsx(
        "relative overflow-hidden transition-all hover:shadow-lg",
        isRecommended && "ring-2 ring-green-500 ring-offset-2",
      )}
    >
      {/* Рекомендуемый бейдж */}
      {isRecommended && (
        <div className="absolute top-0 right-0">
          <div className="bg-green-500 text-white text-xs font-bold px-3 py-1 rounded-bl-lg flex items-center gap-1">
            <SparklesIcon className="w-3 h-3" />
            Рекомендуется
          </div>
        </div>
      )}

      {/* Заголовок */}
      <div className="flex items-start gap-4 mb-4">
        <div
          className={clsx(
            "w-14 h-14 rounded-xl flex items-center justify-center bg-linear-to-br text-white shrink-0",
            display.gradient,
          )}
        >
          <Icon className="w-7 h-7" />
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="text-xl font-bold text-gray-900">{info.name}</h3>
          <p className={clsx("text-sm font-medium", display.color)}>
            {display.tagline}
          </p>
        </div>
      </div>

      {/* Описание */}
      <p className="text-gray-600 mb-4">{info.description}</p>

      {/* Сложность */}
      <div className="grid grid-cols-2 gap-3 mb-4">
        <div className="bg-gray-50 rounded-lg p-3">
          <div className="flex items-center gap-2 text-gray-500 text-xs mb-1">
            <ClockIcon className="w-4 h-4" />
            Время
          </div>
          <code className="text-sm font-mono text-gray-800">
            {info.timeComplexity}
          </code>
        </div>
        <div className="bg-gray-50 rounded-lg p-3">
          <div className="flex items-center gap-2 text-gray-500 text-xs mb-1">
            <CpuChipIcon className="w-4 h-4" />
            Память
          </div>
          <code className="text-sm font-mono text-gray-800">
            {info.spaceComplexity}
          </code>
        </div>
      </div>

      {/* Возможности */}
      {(info.supportsMinCost || info.supportsNegativeCosts) && (
        <div className="flex flex-wrap gap-2 mb-4">
          {info.supportsMinCost && (
            <Badge variant="success">
              <CurrencyDollarIcon className="w-3 h-3 mr-1" />
              Min-Cost Flow
            </Badge>
          )}
          {info.supportsNegativeCosts && (
            <Badge variant="info">
              <ScaleIcon className="w-3 h-3 mr-1" />
              Отрицательные стоимости
            </Badge>
          )}
        </div>
      )}

      {/* Лучше всего подходит для */}
      {info.bestFor && info.bestFor.length > 0 && (
        <div className="border-t border-gray-100 pt-4">
          <p className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
            Лучше всего подходит для:
          </p>
          <div className="flex flex-wrap gap-2">
            {info.bestFor.map((use) => {
              const labelInfo = BEST_FOR_LABELS[use] ?? {
                label: use.replace(/_/g, " "),
                icon: "✓",
              };
              return (
                <span
                  key={use}
                  className="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 rounded-md text-xs text-gray-700"
                >
                  <span>{labelInfo.icon}</span>
                  {labelInfo.label}
                </span>
              );
            })}
          </div>
        </div>
      )}
    </Card>
  );
}

// ============================================================================
// Таблица сравнения
// ============================================================================

interface ComparisonTableProps {
  algorithms: AlgorithmInfo[];
}

function ComparisonTable({ algorithms }: ComparisonTableProps) {
  return (
    <Card>
      <h2 className="text-lg font-semibold mb-4">Сравнительная таблица</h2>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="text-left py-3 px-4 font-medium text-gray-500">
                Алгоритм
              </th>
              <th className="text-left py-3 px-4 font-medium text-gray-500">
                Временная сложность
              </th>
              <th className="text-left py-3 px-4 font-medium text-gray-500">
                Память
              </th>
              <th className="text-center py-3 px-4 font-medium text-gray-500">
                Min-Cost
              </th>
              <th className="text-center py-3 px-4 font-medium text-gray-500">
                Рекомендуется
              </th>
            </tr>
          </thead>
          <tbody>
            {algorithms.map((algo) => {
              const display = ALGORITHM_DISPLAY[algo.algorithm];
              const isRecommended = algo.algorithm === Algorithm.DINIC;

              return (
                <tr
                  key={algo.algorithm}
                  className={clsx(
                    "border-b border-gray-100 hover:bg-gray-50",
                    isRecommended && "bg-green-50",
                  )}
                >
                  <td className="py-3 px-4">
                    <div className="flex items-center gap-2">
                      <span className={display?.color}>{algo.name}</span>
                    </div>
                  </td>
                  <td className="py-3 px-4">
                    <code className="text-xs bg-gray-100 px-2 py-1 rounded">
                      {algo.timeComplexity}
                    </code>
                  </td>
                  <td className="py-3 px-4">
                    <code className="text-xs bg-gray-100 px-2 py-1 rounded">
                      {algo.spaceComplexity}
                    </code>
                  </td>
                  <td className="py-3 px-4 text-center">
                    {algo.supportsMinCost ? (
                      <CheckCircleIcon className="w-5 h-5 text-green-500 mx-auto" />
                    ) : (
                      <span className="text-gray-300">—</span>
                    )}
                  </td>
                  <td className="py-3 px-4 text-center">
                    {isRecommended ? (
                      <SparklesIcon className="w-5 h-5 text-green-500 mx-auto" />
                    ) : (
                      <span className="text-gray-300">—</span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </Card>
  );
}

// ============================================================================
// Руководство по выбору
// ============================================================================

function SelectionGuide() {
  const scenarios = [
    {
      question: "Вы изучаете алгоритмы потока?",
      answer: "Ford-Fulkerson",
      description:
        "Классический алгоритм, легко понять и реализовать. Идеален для обучения.",
      icon: AcademicCapIcon,
      color: "purple",
    },
    {
      question: "Нужен надёжный универсальный алгоритм?",
      answer: "Edmonds-Karp",
      description:
        "Гарантированная полиномиальная сложность. Хорошо работает на большинстве графов.",
      icon: BeakerIcon,
      color: "blue",
    },
    {
      question: "Важна производительность?",
      answer: "Dinic",
      description:
        "Лучший выбор для продакшена. Быстрый на больших графах и двудольных сопоставлениях.",
      icon: BoltIcon,
      color: "green",
    },
    {
      question: "Очень плотный граф с миллионами рёбер?",
      answer: "Push-Relabel",
      description:
        "Оптимален для плотных графов. Использует локальные операции вместо поиска путей.",
      icon: CpuChipIcon,
      color: "orange",
    },
    {
      question: "Нужно минимизировать стоимость доставки?",
      answer: "Min-Cost Flow",
      description:
        "Единственный алгоритм, учитывающий стоимость рёбер. Идеален для логистики.",
      icon: CurrencyDollarIcon,
      color: "emerald",
    },
  ];

  return (
    <Card>
      <h2 className="text-lg font-semibold mb-4">Какой алгоритм выбрать?</h2>
      <div className="space-y-4">
        {scenarios.map((scenario, index) => (
          <div
            key={index}
            className={clsx(
              "flex gap-4 p-4 rounded-lg border-l-4",
              `border-${scenario.color}-500 bg-${scenario.color}-50`,
            )}
            style={{
              borderLeftColor: `var(--color-${scenario.color}-500, #10b981)`,
              backgroundColor: `var(--color-${scenario.color}-50, #ecfdf5)`,
            }}
          >
            <div
              className={`w-10 h-10 rounded-lg bg-${scenario.color}-100 flex items-center justify-center shrink-0`}
              style={{
                backgroundColor: `var(--color-${scenario.color}-100, #d1fae5)`,
              }}
            >
              <scenario.icon
                className={`w-5 h-5 text-${scenario.color}-600`}
                style={{ color: `var(--color-${scenario.color}-600, #059669)` }}
              />
            </div>
            <div>
              <p className="text-gray-600 text-sm">{scenario.question}</p>
              <p className="font-bold text-gray-900">→ {scenario.answer}</p>
              <p className="text-sm text-gray-500 mt-1">
                {scenario.description}
              </p>
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

// ============================================================================
// Главная страница
// ============================================================================

export default function Algorithms() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["algorithms"],
    queryFn: () => solverService.getAlgorithms(),
    staleTime: Infinity,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error) {
    return (
      <Card className="bg-red-50 border-red-200">
        <div className="flex items-center gap-3">
          <ExclamationTriangleIcon className="w-6 h-6 text-red-500" />
          <div>
            <p className="text-red-800 font-medium">
              Ошибка загрузки алгоритмов
            </p>
            <p className="text-red-600 text-sm">{(error as Error).message}</p>
          </div>
        </div>
      </Card>
    );
  }

  const algorithms = data?.algorithms ?? [];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            Алгоритмы максимального потока
          </h1>
          <p className="text-gray-500 mt-1">
            Выберите оптимальный алгоритм для вашей задачи
          </p>
        </div>
        <Link to="/network">
          <Button>
            Перейти к редактору
            <ArrowRightIcon className="w-4 h-4 ml-1" />
          </Button>
        </Link>
      </div>

      {/* Руководство по выбору */}
      <SelectionGuide />

      {/* Карточки алгоритмов */}
      <div>
        <h2 className="text-lg font-semibold mb-4">Доступные алгоритмы</h2>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {algorithms.map((algo) => (
            <AlgorithmCard
              key={algo.algorithm}
              info={algo}
              isRecommended={algo.algorithm === Algorithm.DINIC}
            />
          ))}
        </div>
      </div>

      {/* Таблица сравнения */}
      <ComparisonTable algorithms={algorithms} />

      {/* CTA */}
      <Card className="bg-linear-to-r from-primary-500 to-primary-600 text-white">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-xl font-bold">Готовы попробовать?</h3>
            <p className="text-primary-100 mt-1">
              Создайте граф и протестируйте любой алгоритм
            </p>
          </div>
          <Link to="/network">
            <Button
              variant="secondary"
              className="bg-white text-primary-600 hover:bg-primary-50"
            >
              Открыть редактор
              <ArrowRightIcon className="w-4 h-4 ml-1" />
            </Button>
          </Link>
        </div>
      </Card>
    </div>
  );
}
