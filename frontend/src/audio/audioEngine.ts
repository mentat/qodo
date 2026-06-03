import type { Track } from '../types/radio';

// A single shared <audio> element + Web Audio graph so the Radio app and the
// chat-panel spectrum analyzer can both read the same AnalyserNode. The
// AudioContext is created lazily inside a user gesture (autoplay policy).

let audioEl: HTMLAudioElement | null = null;
let ctx: AudioContext | null = null;
let analyser: AnalyserNode | null = null;
let srcNode: MediaElementAudioSourceNode | null = null;

function getEl(): HTMLAudioElement {
  if (!audioEl) {
    audioEl = new Audio();
    // NOTE: we deliberately do NOT set crossOrigin='anonymous'. The default
    // synthwave tracks (and most free hosts) don't send CORS headers, and
    // crossOrigin would block playback entirely. The trade-off: the
    // AnalyserNode reads a "tainted" (silent) stream, so the visualizer falls
    // back to a synthetic animation while playing. To get TRUE audio-reactive
    // bars, point RADIO_TRACKS at CORS-enabled (Access-Control-Allow-Origin)
    // audio and re-enable crossOrigin here.
    audioEl.preload = 'none';
  }
  return audioEl;
}

function ensureGraph(): void {
  if (!ctx) {
    const Ctor =
      window.AudioContext ||
      (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
    ctx = new Ctor();
    analyser = ctx.createAnalyser();
    analyser.fftSize = 128; // 64 frequency bins — plenty for a bar visualizer
    // createMediaElementSource may only be called once per element.
    srcNode = ctx.createMediaElementSource(getEl());
    srcNode.connect(analyser);
    analyser.connect(ctx.destination);
  }
  if (ctx.state === 'suspended') void ctx.resume();
}

export async function play(track: Track): Promise<void> {
  const el = getEl();
  ensureGraph();
  if (el.src !== track.url) el.src = track.url;
  await el.play();
}

export function pause(): void {
  audioEl?.pause();
}

export function getAnalyser(): AnalyserNode | null {
  return analyser;
}

export function getAudioElement(): HTMLAudioElement {
  return getEl();
}
