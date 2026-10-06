import type { PriorityPresentation } from '@/presentation/priorityConfig';

export function PriorityLabel({ presentation }: { presentation: PriorityPresentation }) {
  const { label, symbol, description } = presentation;
  return (
    <span className="problem-priority-label" title={description}>
      {symbol ? <span aria-hidden="true">{symbol} </span> : null}
      {label}
    </span>
  );
}
