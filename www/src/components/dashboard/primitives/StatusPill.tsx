export interface StatusPillProps {
  label: string;
  color?: string; // hex color
  bgColor?: string; // rgba bg
}

export const StatusPill = ({ label, color = '#22C55E', bgColor = 'rgba(34, 197, 94, 0.1)' }: StatusPillProps) => {
  return (
    <span
      className="text-[9px] font-medium px-2 py-0.5 rounded-[10px]"
      style={{
        color,
        backgroundColor: bgColor,
      }}
    >
      {label}
    </span>
  );
};
