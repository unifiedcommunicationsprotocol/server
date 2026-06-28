export interface MethodBadgeProps {
  method: 'GET' | 'POST';
}

export const MethodBadge = ({ method }: MethodBadgeProps) => {
  const isGet = method === 'GET';
  const bgColor = isGet ? 'bg-[rgba(34,197,94,0.08)]' : 'bg-[rgba(99,102,241,0.12)]';
  const textColor = isGet ? 'text-[#22C55E]' : 'text-[#6366F1]';

  return (
    <span className={`text-[9px] font-semibold font-mono px-[5px] py-[1px] rounded-[3px] shrink-0 ${bgColor} ${textColor}`}>
      {method}
    </span>
  );
};
