// Plays Marvin's synthesized voice (base64 MP3 from the chat response). A
// single dedicated <audio> element — separate from the radio's audioEngine —
// so a new reply interrupts the previous one cleanly.
let el: HTMLAudioElement | null = null;

export function playVoice(base64: string, mime: string): void {
  stopVoice();
  el = new Audio(`data:${mime};base64,${base64}`);
  // Best-effort: if the browser blocks autoplay (no prior gesture), ignore it.
  el.play().catch(() => undefined);
}

export function stopVoice(): void {
  if (el) {
    el.pause();
    el = null;
  }
}
