import type { ReactNode } from 'react';

export interface DataBlockProps {
  label: string;
  value: ReactNode;
  valueColor?: string;
  mono?: boolean;
}

export const DataBlock = ({ label, value, valueColor = '#A1A1AA', mono = false }: DataBlockProps) => {
  return (
    <div className="bg-[#09090B] border border-[#1E1E22] rounded-md p-[11px]">
      <div className="text-[9px] text-[#52525B] uppercase tracking-[0.06em] mb-1">{label}</div>
      <div className={`text-[11px] ${mono ? 'font-mono' : ''}`} style={{ color: valueColor }}>
        {value}
      </div>
    </div>
  );
};
