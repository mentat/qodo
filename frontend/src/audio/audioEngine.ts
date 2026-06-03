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
    audioEl.crossOrigin = 'anonymous'; // required for AnalyserNode on cross-origin audio
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
