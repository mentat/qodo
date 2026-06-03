// Marvin as an animated SVG robot. Idle: gentle float, slow blink, pulsing
// antenna, flat "mouth". Thinking: faster float/blink, rapid antenna pulse,
// and the mouth bars animate like an equalizer. All motion is CSS (see
// app.css) and respects prefers-reduced-motion.
export function MarvinRobot({ thinking }: { thinking: boolean }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '6px 0 2px' }}>
      <svg
        className={`marvin-robot${thinking ? ' thinking' : ''}`}
        width="124"
        height="120"
        viewBox="0 0 120 120"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        role="img"
        aria-label={thinking ? 'Marvin thinking' : 'Marvin'}
      >
        <defs>
          <linearGradient id="marvinHead" x1="0" y1="0" x2="120" y2="120" gradientUnits="userSpaceOnUse">
            <stop offset="0" stopColor="#9B5DE5" />
            <stop offset="1" stopColor="#00E5FF" />
          </linearGradient>
        </defs>

        {/* antenna */}
        <line x1="60" y1="24" x2="60" y2="11" stroke="url(#marvinHead)" strokeWidth="3" strokeLinecap="round" />
        <circle className="marvin-antenna-tip" cx="60" cy="8" r="4.5" fill="#FF2E97" />

        {/* ears */}
        <rect x="17" y="50" width="7" height="20" rx="3" fill="url(#marvinHead)" />
        <rect x="96" y="50" width="7" height="20" rx="3" fill="url(#marvinHead)" />

        {/* head */}
        <rect x="26" y="24" width="68" height="62" rx="16" fill="#0e0e1a" stroke="url(#marvinHead)" strokeWidth="3" />

        {/* eyes */}
        <circle className="marvin-eye" cx="46" cy="49" r="7" fill="#39FF14" />
        <circle className="marvin-eye marvin-eye-2" cx="74" cy="49" r="7" fill="#39FF14" />

        {/* mouth / equalizer */}
        <rect x="40" y="63" width="40" height="15" rx="5" fill="#06060f" stroke="url(#marvinHead)" strokeWidth="1.5" />
        <g className="marvin-mouth">
          <rect className="marvin-bar" x="46" y="67" width="4" height="8" rx="2" fill="#00E5FF" />
          <rect className="marvin-bar" x="53" y="67" width="4" height="8" rx="2" fill="#39FF14" />
          <rect className="marvin-bar" x="60" y="67" width="4" height="8" rx="2" fill="#FF2E97" />
          <rect className="marvin-bar" x="67" y="67" width="4" height="8" rx="2" fill="#39FF14" />
          <rect className="marvin-bar" x="74" y="67" width="4" height="8" rx="2" fill="#00E5FF" />
        </g>
      </svg>
    </div>
  );
}
