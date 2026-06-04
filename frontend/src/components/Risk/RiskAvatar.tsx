import { useEffect, useRef, useState, type CSSProperties } from 'react';
import { DotLottieReact, type DotLottie } from '@lottiefiles/dotlottie-react';
import type { General } from './cast';

export type RiskAvatarMood = 'none' | 'nod' | 'shake' | 'startled';

interface RiskAvatarFrameProps {
  general: General;
  size?: number;
  active?: boolean;
  thinking?: boolean;
  mood?: RiskAvatarMood;
  moodKey?: number | string;
  className?: string;
}

interface RiskAvatarProps extends RiskAvatarFrameProps {
  hovered: boolean;
}

export function RiskAvatarFrame({
  general,
  size = 58,
  active = false,
  thinking = false,
  mood = 'none',
  moodKey,
  className,
}: RiskAvatarFrameProps) {
  const [hovered, setHovered] = useState(false);
  const style = {
    '--risk-avatar-size': `${size}px`,
    '--risk-avatar-accent': `var(--mantine-color-${general.color}-6)`,
    '--risk-avatar-accent-soft': `var(--mantine-color-${general.color}-light)`,
  } as CSSProperties;
  const classes = [
    'risk-avatar-frame',
    active ? 'is-active' : '',
    thinking ? 'is-thinking' : '',
    className ?? '',
  ].filter(Boolean).join(' ');

  return (
    <span
      className={classes}
      style={style}
      tabIndex={0}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onFocus={() => setHovered(true)}
      onBlur={() => setHovered(false)}
    >
      <RiskAvatar
        general={general}
        size={size}
        active={active}
        thinking={thinking}
        hovered={hovered}
        mood={mood}
        moodKey={moodKey}
      />
    </span>
  );
}

function RiskAvatar({
  general,
  active = false,
  thinking = false,
  hovered,
  mood = 'none',
  moodKey,
}: RiskAvatarProps) {
  const [dotLottie, setDotLottie] = useState<DotLottie | null>(null);
  const [ready, setReady] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);
  const reducedMotion = usePrefersReducedMotion();
  const clearMoodTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!dotLottie) return;
    const onReady = () => {
      setReady(true);
      setLoadFailed(false);
    };
    const onError = () => {
      setLoadFailed(true);
      setReady(false);
    };
    dotLottie.addEventListener('ready', onReady);
    dotLottie.addEventListener('load', onReady);
    dotLottie.addEventListener('loadError', onError);
    dotLottie.addEventListener('renderError', onError);
    dotLottie.addEventListener('stateMachineError', onError);
    return () => {
      dotLottie.removeEventListener('ready', onReady);
      dotLottie.removeEventListener('load', onReady);
      dotLottie.removeEventListener('loadError', onError);
      dotLottie.removeEventListener('renderError', onError);
      dotLottie.removeEventListener('stateMachineError', onError);
    };
  }, [dotLottie]);

  useEffect(() => {
    if (!dotLottie || !ready || reducedMotion) return;
    dotLottie.stateMachineStart();
    dotLottie.stateMachineSetBooleanInput('isActive', active);
    dotLottie.stateMachineSetBooleanInput('isThinking', thinking);
    dotLottie.stateMachineSetBooleanInput('isHovered', hovered);
  }, [active, dotLottie, hovered, ready, reducedMotion, thinking]);

  useEffect(() => {
    if (!dotLottie || !ready || reducedMotion) return;
    if (clearMoodTimer.current) clearTimeout(clearMoodTimer.current);

    if (moodKey === undefined || moodKey === null || mood === 'none') {
      dotLottie.stateMachineSetStringInput('eventMood', 'none');
      return;
    }

    dotLottie.stateMachineSetStringInput('eventMood', mood);
    clearMoodTimer.current = setTimeout(() => {
      dotLottie.stateMachineSetStringInput('eventMood', 'none');
    }, 1200);

    return () => {
      if (clearMoodTimer.current) clearTimeout(clearMoodTimer.current);
    };
  }, [dotLottie, mood, moodKey, ready, reducedMotion]);

  useEffect(() => () => {
    if (clearMoodTimer.current) clearTimeout(clearMoodTimer.current);
  }, []);

  if (loadFailed || !general.avatarSrc) {
    return <RiskAvatarFallback general={general} />;
  }

  return (
    <DotLottieReact
      className="risk-avatar-canvas"
      src={general.avatarSrc}
      stateMachineId={reducedMotion ? undefined : 'risk-avatar'}
      animationId={reducedMotion ? 'idle' : undefined}
      autoplay={!reducedMotion}
      loop={!reducedMotion}
      aria-label={general.avatarAlt}
      data-testid="risk-avatar-lottie"
      renderConfig={{ autoResize: true, devicePixelRatio: 1.5, freezeOnOffscreen: true }}
      dotLottieRefCallback={setDotLottie}
    />
  );
}

function RiskAvatarFallback({ general }: { general: General }) {
  return (
    <span className="risk-avatar-fallback" role="img" aria-label={general.avatarAlt}>
      {general.emoji}
    </span>
  );
}

function usePrefersReducedMotion(): boolean {
  const query = '(prefers-reduced-motion: reduce)';
  const [reducedMotion, setReducedMotion] = useState(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return false;
    return window.matchMedia(query).matches;
  });

  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;
    const media = window.matchMedia(query);
    const onChange = () => setReducedMotion(media.matches);
    onChange();
    media.addEventListener('change', onChange);
    return () => media.removeEventListener('change', onChange);
  }, []);

  return reducedMotion;
}
