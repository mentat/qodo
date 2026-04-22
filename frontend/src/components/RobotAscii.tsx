import { useEffect, useState } from 'react';

// Three idle frames that cycle on a timer. When `thinking` is true, the
// frames rotate faster and random non-structural characters get flipped
// each tick, producing a "glitching" effect in the style of a CRT in
// distress — entirely in the browser, no canvas or images.

const IDLE_FRAMES = [
  String.raw`   _____________
  |  .-------.  |
  | | O  _  O | |
  | |   ---   | |
  |  '-------'  |
  |___[MARVIN]__|
    |  |  |  |
   '-''-''-''-'`,
  String.raw`   _____________
  |  .-------.  |
  | | -  _  - | |
  | |   ~~~   | |
  |  '-------'  |
  |___[MARVIN]__|
    |  |  |  |
   '-''-''-''-'`,
  String.raw`   _____________
  |  .-------.  |
  | | @  _  @ | |
  | |   ___   | |
  |  '-------'  |
  |___[ BZZT ]__|
    |  |  |  |
   '-''-''-''-'`,
];

const GLITCH_CHARS = ['#', '%', '?', '!', '*', '~', '/', '\\', '|'];
const NON_STRUCT = /[OoI\-~@_]/g;

function glitch(s: string, rate: number): string {
  return s.replace(NON_STRUCT, (c) => (Math.random() < rate ? GLITCH_CHARS[Math.floor(Math.random() * GLITCH_CHARS.length)] : c));
}

export function RobotAscii({ thinking }: { thinking: boolean }) {
  const [frame, setFrame] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setFrame((f) => (f + 1) % IDLE_FRAMES.length);
    }, thinking ? 150 : 600);
    return () => clearInterval(interval);
  }, [thinking]);

  const base = IDLE_FRAMES[frame];
  const rendered = thinking ? glitch(base, 0.08) : base;

  return (
    <pre
      style={{
        color: '#00ff88',
        textShadow: '0 0 2px #00ff88, 0 0 6px #00ff8844',
        background: '#0a0a0a',
        padding: '8px 12px',
        borderRadius: 6,
        fontSize: 11,
        lineHeight: 1.2,
        margin: 0,
        textAlign: 'center',
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      }}
    >
      {rendered}
    </pre>
  );
}
