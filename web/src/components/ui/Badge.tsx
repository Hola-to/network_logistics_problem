import clsx from "clsx";

interface BadgeProps {
  children: React.ReactNode;
  variant?: "default" | "success" | "warning" | "error" | "info";
  size?: "sm" | "md";
}

export default function Badge({
  children,
  variant = "default",
  size = "md",
}: BadgeProps) {
  const variants = {
    default: "bg-gray-100 text-gray-800",
    success: "badge-success",
    warning: "badge-warning",
    error: "badge-error",
    info: "badge-info",
  };

  const sizes = {
    sm: "px-2 py-0.5 text-xs",
    md: "px-2.5 py-0.5 text-xs",
  };

  return (
    <span className={clsx("badge", variants[variant], sizes[size])}>
      {children}
    </span>
  );
}
