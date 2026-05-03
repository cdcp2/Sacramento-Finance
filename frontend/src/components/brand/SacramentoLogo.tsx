import type { CSSProperties } from 'react'
import sacramentoMark from '@/assets/sacramento-mark.png'

type SacramentoLogoSize = 'sm' | 'md' | 'lg'

type SacramentoLogoProps = {
  size?: SacramentoLogoSize
  showText?: boolean
}

const sizeConfig: Record<
  SacramentoLogoSize,
  { logoSize: string; nameClass: string; subClass: string; gapClass: string }
> = {
  sm: { logoSize: '48px', nameClass: 'text-[14px]', subClass: 'text-[7.5px]', gapClass: 'gap-1.5' },
  md: { logoSize: '66px', nameClass: 'text-[18px]', subClass: 'text-[9px]', gapClass: 'gap-2' },
  lg: { logoSize: '88px', nameClass: 'text-[22px]', subClass: 'text-[10px]', gapClass: 'gap-2.5' },
}

export default function SacramentoLogo({ size = 'md', showText = true }: SacramentoLogoProps) {
  const cfg = sizeConfig[size]

  return (
    <div className={`sacramento-brand flex items-center ${cfg.gapClass} select-none`}>
      <div
        className="sacramento-mark"
        style={{ '--mark-size': cfg.logoSize } as CSSProperties}
        aria-hidden="true"
      >
        <img className="sacramento-mark__image" src={sacramentoMark} alt="" />
      </div>

      {showText && (
        <div className="leading-none">
          <p className={`font-bold text-white tracking-wide ${cfg.nameClass}`}>Sacramento</p>
          <div className="mt-0.5 flex items-center gap-1">
            <span className={`${cfg.subClass} text-brand-cyan`}>-</span>
            <span className={`${cfg.subClass} font-semibold tracking-[0.18em] text-brand-cyan`}>
              FINANCE
            </span>
            <span className={`${cfg.subClass} text-brand-cyan`}>-</span>
          </div>
        </div>
      )}
    </div>
  )
}
