import type { ReactNode } from 'react';

export interface SectionCardProps {
  title: string;
  subtitle?: string;
  children: ReactNode;
}

export const SectionCard = ({ title, subtitle, children }: SectionCardProps) => {
  return (
    <div className="bg-[#111113] border border-[#1E1E22] rounded-lg p-[18px]">
      <div className="text-[12px] font-semibold text-[#FAFAFA] mb-1">{title}</div>
      {subtitle && <div className="text-[11px] text-[#52525B] mb-[14px]">{subtitle}</div>}
      {children}
    </div>
  );
};
