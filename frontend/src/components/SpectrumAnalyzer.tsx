import { useEffect, useRef } from 'react';
import { getAnalyser } from '../audio/audioEngine';
import { useRadioStore } from '../store/radioStore';

// A neon bar visualizer for Marvin. When the Synthwave Radio is playing it's
// REAL audio-reactive (Web Audio AnalyserNode); otherwise it draws synthetic
// bars driven by Marvin's state — a gentle idle wave, spiky while thinking.
export function SpectrumAnalyzer({ thinking }: { thinking: boolean }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const playing = useRadioStore((s) => s.playing);
  // Mirror latest props/state into refs so the rAF loop (set up once) sees them.
  const thinkingRef = useRef(thinking);
  const playingRef = useRef(playing);
  thinkingRef.current = thinking;
  playingRef.current = playing;

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let raf = 0;
    let t = 0;
    const BARS = 32;
    const freq = new Uint8Array(64);

    const resize = () => {
      const dpr = window.devicePixelRatio || 1;
      canvas.width = Math.max(1, Math.floor(canvas.clientWidth * dpr));
      canvas.height = Math.max(1, Math.floor(canvas.clientHeight * dpr));
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    };
    const ro = new ResizeObserver(resize);
    ro.observe(canvas);
    resize();

    const draw = () => {
      raf = requestAnimationFrame(draw);
      t += 0.05;
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      ctx.clearRect(0, 0, w, h);

      const analyser = getAnalyser();
      const values: number[] = [];
      if (analyser && playingRef.current) {
        analyser.getByteFrequencyData(freq);
        const step = Math.floor(freq.length / BARS) || 1;
        for (let i = 0; i < BARS; i++) values.push(freq[i * step] / 255);
      } else {
        const amp = thinkingRef.current ? 0.85 : 0.3;
        const speed = thinkingRef.current ? 1.7 : 0.6;
        for (let i = 0; i < BARS; i++) {
          const base = Math.sin(t * speed + i * 0.5) * 0.5 + 0.5;
          const jitter = thinkingRef.current ? Math.abs(Math.sin(t * 5 + i * 1.3)) * 0.4 : 0;
          values.push(Math.min(1, base * amp + jitter));
        }
      }

      const gap = 2;
      const barW = (w - gap * (BARS - 1)) / BARS;
      const grad = ctx.createLinearGradient(0, h, 0, 0);
      grad.addColorStop(0, '#FF2E97');
      grad.addColorStop(0.5, '#9B5DE5');
      grad.addColorStop(1, '#00E5FF');
      ctx.fillStyle = grad;
      ctx.shadowColor = '#9B5DE5';
      ctx.shadowBlur = 8;
      for (let i = 0; i < BARS; i++) {
        const bh = Math.max(2, values[i] * (h - 4));
        ctx.fillRect(i * (barW + gap), h - bh, barW, bh);
      }
    };
    draw();

    return () => {
      cancelAnimationFrame(raf);
      ro.disconnect();
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      aria-label="Marvin spectrum analyzer"
      style={{ width: '100%', height: 72, display: 'block', background: '#0a0a12', borderRadius: 6 }}
    />
  );
}
